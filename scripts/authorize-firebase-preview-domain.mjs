import { execFileSync } from "node:child_process";
import { pathToFileURL } from "node:url";

export function previewHostname(projectId, previewUrl) {
  if (!projectId) throw new Error("GCP_PROJECT_ID is required");

  const url = new URL(previewUrl);
  const hostname = url.hostname.toLowerCase();
  if (url.protocol !== "https:" || !hostname.startsWith(`${projectId}--`) || !hostname.endsWith(".web.app")) {
    throw new Error("FIREBASE_PREVIEW_URL must be a Firebase Hosting preview URL for GCP_PROJECT_ID");
  }
  return hostname;
}

export function mergeAuthorizedDomains(existing, hostname, projectId) {
  const previewPrefix = `${projectId}--pr-`;
  const permanentDomains = (Array.isArray(existing) ? existing : []).filter(
    (domain) => !(domain.startsWith(previewPrefix) && domain.endsWith(".web.app")),
  );
  return [...new Set([...permanentDomains, hostname])].sort();
}

async function request(url, options, requiredPermission) {
  const response = await fetch(url, options);
  if (!response.ok) {
    const detail = (await response.text()).slice(0, 1000);
    if (response.status === 403) {
      throw new Error(
        `Identity Toolkit config request requires ${requiredPermission}; apply the staging OpenTofu IAM changes and retry: ${detail}`,
      );
    }
    throw new Error(`Identity Toolkit config request failed (${response.status}): ${detail}`);
  }
  return response.json();
}

function wait(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function readAuthorizedDomains(endpoint, headers) {
  const config = await request(endpoint, { headers }, "firebaseauth.configs.get");
  return Array.isArray(config.authorizedDomains) ? config.authorizedDomains : [];
}

async function verifyAuthorizedDomain(endpoint, headers, hostname) {
  for (let attempt = 1; attempt <= 6; attempt += 1) {
    const authorizedDomains = await readAuthorizedDomains(endpoint, headers);
    if (authorizedDomains.includes(hostname)) {
      console.log(`Firebase Auth authorization verified for ${hostname}`);
      return;
    }
    if (attempt < 6) {
      await wait(attempt * 1000);
    }
  }
  throw new Error(`Firebase Auth authorization did not verify for ${hostname}`);
}

async function main() {
  const projectId = process.env.GCP_PROJECT_ID;
  const hostname = previewHostname(projectId, process.env.FIREBASE_PREVIEW_URL);
  const token = execFileSync("gcloud", ["auth", "print-access-token"], { encoding: "utf8" }).trim();
  const endpoint = `https://identitytoolkit.googleapis.com/admin/v2/projects/${encodeURIComponent(projectId)}/config`;
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
  const existingDomains = await readAuthorizedDomains(endpoint, headers);
  const authorizedDomains = mergeAuthorizedDomains(existingDomains, hostname, projectId);

  if (JSON.stringify([...existingDomains].sort()) === JSON.stringify(authorizedDomains)) {
    console.log(`Firebase Auth already authorizes ${hostname}`);
    await verifyAuthorizedDomain(endpoint, headers, hostname);
    return;
  }

  await request(
    `${endpoint}?updateMask=authorizedDomains`,
    {
      method: "PATCH",
      headers,
      body: JSON.stringify({ name: `projects/${projectId}/config`, authorizedDomains }),
    },
    "firebaseauth.configs.update",
  );
  console.log(`Firebase Auth now authorizes ${hostname}`);
  await verifyAuthorizedDomain(endpoint, headers, hostname);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
