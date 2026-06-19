import { appendFileSync, readFileSync } from "node:fs";

const filePath = process.argv[2] || "firebase-preview.json";
const raw = readFileSync(filePath, "utf8");
const parsed = JSON.parse(raw);
const urls = [];

function collectUrls(value) {
  if (!value) return;
  if (typeof value === "string") {
    if (/^https:\/\/[^\s"']+$/.test(value)) urls.push(value);
    return;
  }
  if (Array.isArray(value)) {
    value.forEach(collectUrls);
    return;
  }
  if (typeof value === "object") {
    Object.values(value).forEach(collectUrls);
  }
}

collectUrls(parsed);

const hostingUrls = urls.filter((candidate) => {
  try {
    const host = new URL(candidate).hostname;
    return host.endsWith(".web.app") || host.endsWith(".firebaseapp.com");
  } catch {
    return false;
  }
});

const preferred =
  hostingUrls.find((candidate) => /--pr-\d+[-.]/.test(candidate)) ||
  hostingUrls.find((candidate) => /pr-\d+/.test(candidate)) ||
  hostingUrls[0];

if (!preferred) {
  throw new Error("No Firebase Hosting preview URL found in firebase-preview.json");
}

console.error(`Using Firebase Hosting preview URL: ${preferred}`);

if (process.env.GITHUB_OUTPUT) {
  appendFileSync(process.env.GITHUB_OUTPUT, `url=${preferred}\n`);
} else {
  console.log(preferred);
}
