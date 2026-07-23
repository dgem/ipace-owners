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
values.INSTAGRAM_USER_ID = process.env.INSTAGRAM_USER_ID || "";
values.INSTAGRAM_GRAPH_API_VERSION = process.env.INSTAGRAM_GRAPH_API_VERSION || "";
values.INSTAGRAM_MEDIA_BASE_URL = process.env.INSTAGRAM_MEDIA_BASE_URL || "";
values.CAMPAIGN_MEDIA_BUCKET = process.env.CAMPAIGN_MEDIA_BUCKET || "";
values.VEO_LOCATION = process.env.VEO_LOCATION || "";
values.VEO_MODEL_ID = process.env.VEO_MODEL_ID || "";
try {
  new URL(values.FIREBASE_EMAIL_CONTINUE_URL);
} catch {
  throw new Error("FIREBASE_EMAIL_CONTINUE_URL must be an absolute URL");
}
values.GOOGLE_CLOUD_PROJECT = values.FIREBASE_PROJECT_ID;
values.GCP_PROJECT = values.FIREBASE_PROJECT_ID;

writeFileSync("functions-env.json", `${JSON.stringify(values, null, 2)}\n`);
