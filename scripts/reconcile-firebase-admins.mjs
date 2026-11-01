import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

export function parseAdminUsers(value) {
  let users;
  try {
    users = JSON.parse(String(value || "{}"));
  } catch {
    throw new Error("FIREBASE_ADMIN_USERS_JSON must be a JSON object of labels to email addresses");
  }
  if (!users || Array.isArray(users) || typeof users !== "object") {
    throw new Error("FIREBASE_ADMIN_USERS_JSON must be a JSON object of labels to email addresses");
  }

  const entries = Object.entries(users);
  if (entries.length === 0) {
    throw new Error("Refusing to manage Firebase admins with an empty desired set");
  }

  const seen = new Set();
  for (const [label, rawEmail] of entries) {
    const email = String(rawEmail || "").trim().toLowerCase();
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254) {
      throw new Error(`Firebase admin ${label} has an invalid email address`);
    }
    if (seen.has(email)) throw new Error("Firebase administrator email addresses must be unique");
    seen.add(email);
    users[label] = email;
  }
  return users;
}

export function parseRoleUsers(value) {
  let roles;
  try {
    roles = JSON.parse(String(value || "{}"));
  } catch {
    throw new Error("FIREBASE_ROLE_USERS_JSON must be a JSON object of role names to email lists");
  }
  if (!roles || Array.isArray(roles) || typeof roles !== "object") {
    throw new Error("FIREBASE_ROLE_USERS_JSON must be a JSON object of role names to email lists");
  }
  const normalised = {};
  for (const [role, rawEmails] of Object.entries(roles)) {
    if (!/^[a-z][a-z0-9-]{0,63}$/.test(role) || role === "admin") {
      throw new Error(`Firebase custom role ${role} has an invalid name`);
    }
    if (!Array.isArray(rawEmails)) throw new Error(`Firebase custom role ${role} must map to an email list`);
    const emails = rawEmails.map((value) => String(value || "").trim().toLowerCase());
    if (emails.some((email) => !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) || email.length > 254)) {
      throw new Error(`Firebase custom role ${role} has an invalid email address`);
    }
    if (new Set(emails).size !== emails.length) throw new Error(`Firebase custom role ${role} email addresses must be unique`);
    normalised[role] = emails;
  }
  return normalised;
}

function parseClaims(user) {
  if (!user.customAttributes) return {};
  try {
    const claims = JSON.parse(user.customAttributes);
    if (!claims || Array.isArray(claims) || typeof claims !== "object") throw new Error("invalid object");
    return claims;
  } catch {
    throw new Error(`Firebase user ${user.localId} has invalid custom attributes`);
  }
}

function sortValue(value) {
  if (Array.isArray(value)) return value.map(sortValue);
  if (!value || typeof value !== "object") return value;
  return Object.fromEntries(Object.keys(value).sort().map((key) => [key, sortValue(value[key])]));
}

function stableJson(value) {
  return JSON.stringify(sortValue(value));
}

export function buildAdminUpdates(users, desiredAdminUsers, desiredRoleUsers = {}) {
  const usersByEmail = new Map(users.filter((user) => user.email).map((user) => [user.email.toLowerCase(), user]));
  const missingLabels = Object.entries(desiredAdminUsers)
    .filter(([, email]) => !usersByEmail.has(email))
    .map(([label]) => label);
  if (missingLabels.length) {
    throw new Error(`Configured Firebase administrators do not exist: ${missingLabels.join(", ")}`);
  }
  const desiredUids = new Set(Object.values(desiredAdminUsers).map((email) => usersByEmail.get(email).localId));
  if (desiredUids.size === 0) throw new Error("Refusing to remove the final Firebase administrator");
  const roleUids = new Map();
  for (const [role, emails] of Object.entries(desiredRoleUsers)) {
    for (const email of emails) {
      const user = usersByEmail.get(email);
      if (!user) throw new Error(`Firebase custom role ${role} account does not exist: ${email}`);
      roleUids.set(user.localId, [...(roleUids.get(user.localId) || []), role]);
    }
  }

  const updates = [];
  for (const user of users) {
    const current = parseClaims(user);
    const next = { ...current };
    const managedRoles = Array.isArray(current.managedRoles) ? current.managedRoles.filter((role) => typeof role === "string") : [];
    let roles = Array.isArray(current.roles) ? current.roles.filter((role) => !managedRoles.includes(role)) : [];
    const currentlyAdmin = current.admin === true || roles.includes("admin");
    const shouldBeAdmin = desiredUids.has(user.localId);

    if (shouldBeAdmin) {
      next.admin = true;
    } else if (currentlyAdmin) {
      delete next.admin;
      if (Array.isArray(current.roles)) {
        roles = roles.filter((role) => role !== "admin");
      }
    }

    const assignedRoles = roleUids.get(user.localId) || [];
    const nextRoles = [...new Set([...roles, ...assignedRoles])].sort();
    if (nextRoles.length) next.roles = nextRoles;
    else delete next.roles;
    const nextManagedRoles = Object.keys(desiredRoleUsers).sort();
    if (nextManagedRoles.length) next.managedRoles = nextManagedRoles;
    else delete next.managedRoles;

    if (stableJson(current) !== stableJson(next)) {
      updates.push({ uid: user.localId, customAttributes: JSON.stringify(next), grant: shouldBeAdmin });
    }
  }
  return updates;
}

async function request(url, options, operation) {
  const response = await fetch(url, options);
  const text = await response.text();
  if (!response.ok) throw new Error(`${operation} failed (${response.status}): ${text.slice(0, 1000)}`);
  return text ? JSON.parse(text) : {};
}

async function listUsers(projectId, headers) {
  const users = [];
  let pageToken = "";
  do {
    const endpoint = new URL(
      `https://identitytoolkit.googleapis.com/v1/projects/${encodeURIComponent(projectId)}/accounts:batchGet`,
    );
    endpoint.searchParams.set("maxResults", "1000");
    if (pageToken) endpoint.searchParams.set("nextPageToken", pageToken);
    const page = await request(endpoint, { headers }, "Firebase Auth user listing");
    users.push(...(page.users || []));
    pageToken = page.nextPageToken || "";
  } while (pageToken);
  return users;
}

async function main() {
  const projectId = String(process.env.GCP_PROJECT_ID || "").trim();
  if (!projectId) throw new Error("GCP_PROJECT_ID is required");
  const desiredAdminUsers = parseAdminUsers(process.env.FIREBASE_ADMIN_USERS_JSON);
  const desiredRoleUsers = parseRoleUsers(process.env.FIREBASE_ROLE_USERS_JSON);
  const token = execFileSync("gcloud", ["auth", "print-access-token"], { encoding: "utf8" }).trim();
  const headers = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
    "X-Goog-User-Project": projectId,
  };
  const users = await listUsers(projectId, headers);
  const updates = buildAdminUpdates(users, desiredAdminUsers, desiredRoleUsers);

  for (const update of updates) {
    await request(
      "https://identitytoolkit.googleapis.com/v1/accounts:update",
      {
        method: "POST",
        headers,
        body: JSON.stringify({
          localId: update.uid,
          targetProjectId: projectId,
          customAttributes: update.customAttributes,
        }),
      },
      `Firebase administrator ${update.grant ? "grant" : "revocation"}`,
    );
  }

  const granted = updates.filter((update) => update.grant).length;
  const revoked = updates.length - granted;
  console.log(`Firebase administrators reconciled for ${projectId}: ${granted} granted, ${revoked} revoked, ${desiredAdminUsers && Object.keys(desiredAdminUsers).length} desired`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
