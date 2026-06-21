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
try {
  values.FIREBASE_EMAIL_LINK_DOMAIN =
    process.env.FIREBASE_EMAIL_LINK_DOMAIN || new URL(values.FIREBASE_EMAIL_CONTINUE_URL).hostname;
} catch {
  throw new Error("FIREBASE_EMAIL_CONTINUE_URL must be an absolute URL");
}
values.GOOGLE_CLOUD_PROJECT = values.FIREBASE_PROJECT_ID;
values.GCP_PROJECT = values.FIREBASE_PROJECT_ID;

writeFileSync("functions-env.json", `${JSON.stringify(values, null, 2)}\n`);
