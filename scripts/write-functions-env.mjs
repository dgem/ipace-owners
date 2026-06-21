import { writeFileSync } from "node:fs";

const required = [
  "FIREBASE_PROJECT_ID",
  "FIRESTORE_DATABASE_ID",
  "FIREBASE_WEB_API_KEY",
  "VIN_PEPPER",
  "SNAPSHOT_BUCKET",
  "ALLOWED_ORIGINS",
  "FIREBASE_EMAIL_CONTINUE_URL",
  "FIREBASE_EMAIL_LINK_DOMAIN",
];

const missing = required.filter((name) => !process.env[name]);

if (missing.length) {
  throw new Error(`Missing required function environment values: ${missing.join(", ")}`);
}

const values = Object.fromEntries(required.map((name) => [name, process.env[name]]));
values.GOOGLE_CLOUD_PROJECT = values.FIREBASE_PROJECT_ID;
values.GCP_PROJECT = values.FIREBASE_PROJECT_ID;

writeFileSync("functions-env.json", `${JSON.stringify(values, null, 2)}\n`);
