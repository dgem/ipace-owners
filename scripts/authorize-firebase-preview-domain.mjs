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

async function request(url, options) {
  const response = await fetch(url, options);
  if (!response.ok) {
    const detail = (await response.text()).slice(0, 1000);
    throw new Error(`Identity Toolkit config request failed (${response.status}): ${detail}`);
  }
  return response.json();
}

async function main() {
  const projectId = process.env.GCP_PROJECT_ID;
  const hostname = previewHostname(projectId, process.env.FIREBASE_PREVIEW_URL);
  const token = execFileSync("gcloud", ["auth", "print-access-token"], { encoding: "utf8" }).trim();
  const endpoint = `https://identitytoolkit.googleapis.com/admin/v2/projects/${encodeURIComponent(projectId)}/config`;
  const headers = { Authorization: `Bearer ${token}`, "Content-Type": "application/json" };
  const config = await request(endpoint, { headers });
  const authorizedDomains = mergeAuthorizedDomains(config.authorizedDomains, hostname, projectId);

  const existingDomains = [...(config.authorizedDomains || [])].sort();
  if (JSON.stringify(existingDomains) === JSON.stringify(authorizedDomains)) {
    console.log(`Firebase Auth already authorizes ${hostname}`);
    return;
  }

  await request(`${endpoint}?updateMask=authorizedDomains`, {
    method: "PATCH",
    headers,
    body: JSON.stringify({ name: `projects/${projectId}/config`, authorizedDomains }),
  });
  console.log(`Firebase Auth now authorizes ${hostname}`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await main();
}
