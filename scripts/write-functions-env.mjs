import { writeFileSync } from "node:fs";

const required = [
  "FIREBASE_PROJECT_ID",
  "FIREBASE_WEB_API_KEY",
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
values.RESEND_FROM = process.env.RESEND_FROM || "";
values.RESEND_REPLY_TO = process.env.RESEND_REPLY_TO || "";
values.RESEND_ASSET_BASE_URL = process.env.RESEND_ASSET_BASE_URL || "";
values.INSTAGRAM_USER_ID = process.env.INSTAGRAM_USER_ID || "";
values.INSTAGRAM_GRAPH_API_VERSION = process.env.INSTAGRAM_GRAPH_API_VERSION || "";
values.INSTAGRAM_MEDIA_BASE_URL = process.env.INSTAGRAM_MEDIA_BASE_URL || "";
values.CAMPAIGN_MEDIA_BUCKET = process.env.CAMPAIGN_MEDIA_BUCKET || "";
values.MEMBER_STORY_MEDIA_BUCKET = process.env.MEMBER_STORY_MEDIA_BUCKET || "";
try {
  const featureAccess = JSON.parse(process.env.MEMBER_FEATURE_ACCESS || "{}");
  if (!featureAccess || Array.isArray(featureAccess) || typeof featureAccess !== "object") throw new Error();
  values.MEMBER_FEATURE_ACCESS = JSON.stringify(featureAccess);
} catch {
  throw new Error("MEMBER_FEATURE_ACCESS must be a JSON object");
}
values.VEO_LOCATION = process.env.VEO_LOCATION || "";
values.VEO_MODEL_ID = process.env.VEO_MODEL_ID || "";
values.SEED_NEXT_STEPS_SURVEY = process.env.SEED_NEXT_STEPS_SURVEY || "";
values.NEXT_STEPS_SURVEY_STATUS = process.env.NEXT_STEPS_SURVEY_STATUS || "";
try {
  new URL(values.FIREBASE_EMAIL_CONTINUE_URL);
} catch {
  throw new Error("FIREBASE_EMAIL_CONTINUE_URL must be an absolute URL");
}
values.GOOGLE_CLOUD_PROJECT = values.FIREBASE_PROJECT_ID;
values.GCP_PROJECT = values.FIREBASE_PROJECT_ID;
// Signed Cloud Storage URLs need the runtime service account as the
// authorising identity.  The deploy workflow already supplies this value to
// gcloud; copy it into the Functions environment without storing a key.
values.STORAGE_SIGNING_SERVICE_ACCOUNT = process.env.FUNCTIONS_SERVICE_ACCOUNT || "";

writeFileSync("functions-env.json", `${JSON.stringify(values, null, 2)}\n`);
