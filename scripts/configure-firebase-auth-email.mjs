import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

export function normalizeDomain(value, name) {
  const domain = String(value || "").trim().toLowerCase();
  if (!domain) return "";
  if (!/^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,63}$/.test(domain)) {
    throw new Error(`${name} must be a hostname without a scheme, path, or port`);
  }
  return domain;
}

export function buildEmailConfig() {
  const sendEmail = {
    method: "DEFAULT",
  };

  return { notification: { defaultLocale: "en-GB", sendEmail } };
}

export function emailConfigUpdateMask() {
  return [
    "notification.defaultLocale",
    "notification.sendEmail.method",
  ].join(",");
}

export function emailConfigUpdates(config) {
  const sendEmail = config.notification.sendEmail;
  const updates = [
    {
      name: "default locale",
      mask: "notification.defaultLocale",
      payload: { notification: { defaultLocale: config.notification.defaultLocale } },
    },
    {
      name: "email delivery method",
      mask: "notification.sendEmail.method",
      payload: { notification: { sendEmail: { method: sendEmail.method } } },
    },
  ];

  return updates;
}

export function emailDomainVerificationEndpoint(configEndpoint) {
  return `${configEndpoint.replace(/\/config$/, "")}/domain:verify`;
}

async function request(url, options, operation) {
  const response = await fetch(url, options);
  const text = await response.text();
  if (!response.ok) {
    throw new Error(`${operation} failed (${response.status}): ${text.slice(0, 1000)}`);
  }
  return text ? JSON.parse(text) : {};
}

async function reconcileEmailDomain(endpoint, headers, config, emailDomain) {
  if (!emailDomain) return;
  const dnsInfo = config.notification?.sendEmail?.dnsInfo || {};
  if (dnsInfo.customDomain === emailDomain && dnsInfo.useCustomDomain) {
    console.log(`Firebase Auth email sender domain is active: ${emailDomain}`);
    return;
  }

  let state = dnsInfo.customDomainState;
  if (dnsInfo.pendingCustomDomain !== emailDomain || state === "NOT_STARTED" || state === "FAILED") {
    const result = await request(
      emailDomainVerificationEndpoint(endpoint),
      { method: "POST", headers, body: JSON.stringify({ domain: emailDomain, action: "VERIFY" }) },
      "Firebase Auth email-domain verification",
    );
    state = result.verificationState;
  }

  if (state === "SUCCEEDED") {
    await request(
      emailDomainVerificationEndpoint(endpoint),
      { method: "POST", headers, body: JSON.stringify({ domain: emailDomain, action: "APPLY" }) },
      "Firebase Auth email-domain activation",
    );
    console.log(`Firebase Auth email sender domain activated: ${emailDomain}`);
    return;
  }

  console.log(`Firebase Auth email sender domain verification is ${state || "IN_PROGRESS"}: ${emailDomain}`);
  console.log("Add the TXT and CNAME records shown under Firebase Authentication > Templates, then run make infra-email-domain ENV=<environment>.");
}

async function main() {
  const projectId = process.env.GCP_PROJECT_ID;
  if (!projectId) throw new Error("GCP_PROJECT_ID is required");

  const emailDomain = normalizeDomain(process.env.FIREBASE_AUTH_EMAIL_DOMAIN, "FIREBASE_AUTH_EMAIL_DOMAIN");
  const token = execFileSync("gcloud", ["auth", "print-access-token"], { encoding: "utf8" }).trim();
  const endpoint = `https://identitytoolkit.googleapis.com/admin/v2/projects/${encodeURIComponent(projectId)}/config`;
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json", "X-Goog-User-Project": projectId };
  const currentConfig = await request(endpoint, { headers }, "Firebase Auth config read");
  const payload = buildEmailConfig();
  // Domain verification is a separate two-phase API. Attempting to set
  // dnsInfo.useCustomDomain before VERIFY succeeds causes
  // EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED and rejects this whole PATCH.
  for (const update of emailConfigUpdates(payload)) {
    await request(
      `${endpoint}?updateMask=${encodeURIComponent(update.mask)}`,
      { method: "PATCH", headers, body: JSON.stringify(update.payload) },
      `Firebase Auth ${update.name} update`,
    );
  }
  console.log(`Firebase Auth email settings configured for ${projectId}`);
  await reconcileEmailDomain(endpoint, headers, currentConfig, emailDomain);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
