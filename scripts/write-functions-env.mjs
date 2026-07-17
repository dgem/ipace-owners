import { writeFileSync } from "node:fs";

const required = [
  "FIREBASE_PROJECT_ID",
  "FIREBASE_WEB_API_KEY",
  "VIN_PEPPER",
  "SNAPSHOT_BUCKET",
  "ALLOWED_ORIGINS",
  "FIREBASE_EMAIL_CONTINUE_URL",
];

const missing = required.filter((name) => !process.env[name]);

if (missing.length) {
  throw new Error(`Missing required function environment values: ${missing.join(", ")}`);
}

const values = Object.fromEntries(required.map((name) => [name, process.env[name]]));
values.FIRESTORE_DATABASE_ID = process.env.FIRESTORE_DATABASE_ID || values.FIREBASE_PROJECT_ID;
values.FIREBASE_EMAIL_LINK_DOMAIN = process.env.FIREBASE_EMAIL_LINK_DOMAIN || "";
values.RESEND_API_KEY = process.env.RESEND_API_KEY || "";
values.RESEND_FROM = process.env.RESEND_FROM || "";
values.RESEND_REPLY_TO = process.env.RESEND_REPLY_TO || "";
values.RESEND_ASSET_BASE_URL = process.env.RESEND_ASSET_BASE_URL || "";
try {
  new URL(values.FIREBASE_EMAIL_CONTINUE_URL);
} catch {
  throw new Error("FIREBASE_EMAIL_CONTINUE_URL must be an absolute URL");
}
values.GOOGLE_CLOUD_PROJECT = values.FIREBASE_PROJECT_ID;
values.GCP_PROJECT = values.FIREBASE_PROJECT_ID;

writeFileSync("functions-env.json", `${JSON.stringify(values, null, 2)}\n`);
