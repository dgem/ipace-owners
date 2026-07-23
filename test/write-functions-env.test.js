const assert = require("node:assert/strict");
const { execFileSync, spawnSync } = require("node:child_process");
const { mkdtempSync, readFileSync } = require("node:fs");
const { tmpdir } = require("node:os");
const { join, resolve } = require("node:path");
const { test } = require("node:test");

const scriptPath = resolve(__dirname, "../scripts/write-functions-env.mjs");

test("writes function env vars as JSON without splitting comma-separated origins", () => {
  const cwd = mkdtempSync(join(tmpdir(), "ipace-functions-env-"));

  execFileSync(process.execPath, [scriptPath], {
    cwd,
    env: {
      ...process.env,
      FIREBASE_PROJECT_ID: "ipace-owners-staging",
      FIRESTORE_DATABASE_ID: "ipace-owners-staging",
      FIREBASE_WEB_API_KEY: "api-key",
      VIN_PEPPER: "pepper",
      SNAPSHOT_BUCKET: "snapshots",
      ALLOWED_ORIGINS: "https://stage.ipace-owners.org,http://localhost:8080,http://localhost:5000",
      FIREBASE_EMAIL_CONTINUE_URL: "https://stage.ipace-owners.org/member/account/",
      FIREBASE_EMAIL_LINK_DOMAIN: "stage.ipace-owners.org",
      RESEND_API_KEY: "resend-key",
      RESEND_FROM: "I-PACE Owners <members@stage.ipace-owners.org>",
      RESEND_REPLY_TO: "contact@ipace-owners.org",
      RESEND_ASSET_BASE_URL: "https://stage.ipace-owners.org",
      INSTAGRAM_USER_ID: "123456789",
      INSTAGRAM_GRAPH_API_VERSION: "v99.0",
      INSTAGRAM_MEDIA_BASE_URL: "https://stage.ipace-owners.org",
      CAMPAIGN_MEDIA_BUCKET: "ipace-owners-staging-campaign-media",
      VEO_LOCATION: "global",
      VEO_MODEL_ID: "veo-3.1-generate-001",
    },
  });

  const written = JSON.parse(readFileSync(join(cwd, "functions-env.json"), "utf8"));

  assert.deepEqual(written, {
    FIREBASE_PROJECT_ID: "ipace-owners-staging",
    FIRESTORE_DATABASE_ID: "ipace-owners-staging",
    FIREBASE_WEB_API_KEY: "api-key",
    VIN_PEPPER: "pepper",
    SNAPSHOT_BUCKET: "snapshots",
    ALLOWED_ORIGINS: "https://stage.ipace-owners.org,http://localhost:8080,http://localhost:5000",
    FIREBASE_EMAIL_CONTINUE_URL: "https://stage.ipace-owners.org/member/account/",
    FIREBASE_EMAIL_LINK_DOMAIN: "stage.ipace-owners.org",
    RESEND_API_KEY: "resend-key",
    RESEND_FROM: "I-PACE Owners <members@stage.ipace-owners.org>",
    RESEND_REPLY_TO: "contact@ipace-owners.org",
    RESEND_ASSET_BASE_URL: "https://stage.ipace-owners.org",
    INSTAGRAM_USER_ID: "123456789",
    INSTAGRAM_GRAPH_API_VERSION: "v99.0",
    INSTAGRAM_MEDIA_BASE_URL: "https://stage.ipace-owners.org",
    CAMPAIGN_MEDIA_BUCKET: "ipace-owners-staging-campaign-media",
    VEO_LOCATION: "global",
    VEO_MODEL_ID: "veo-3.1-generate-001",
    GOOGLE_CLOUD_PROJECT: "ipace-owners-staging",
    GCP_PROJECT: "ipace-owners-staging",
  });
});

test("fails when required function env vars are missing", () => {
  const cwd = mkdtempSync(join(tmpdir(), "ipace-functions-env-"));
  const result = spawnSync(process.execPath, [scriptPath], {
    cwd,
    env: {
      PATH: process.env.PATH,
    },
    encoding: "utf8",
  });

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /Missing required function environment values/);
});

test("derives the database ID while leaving the preview link domain unset", () => {
  const cwd = mkdtempSync(join(tmpdir(), "ipace-functions-env-"));

  execFileSync(process.execPath, [scriptPath], {
    cwd,
    env: {
      PATH: process.env.PATH,
      FIREBASE_PROJECT_ID: "ipace-owners-staging",
      FIREBASE_WEB_API_KEY: "api-key",
      VIN_PEPPER: "pepper",
      SNAPSHOT_BUCKET: "snapshots",
      ALLOWED_ORIGINS: "https://stage.ipace-owners.org",
      FIREBASE_EMAIL_CONTINUE_URL: "https://stage.ipace-owners.org/member/account/",
    },
  });

  const written = JSON.parse(readFileSync(join(cwd, "functions-env.json"), "utf8"));

  assert.equal(written.FIRESTORE_DATABASE_ID, "ipace-owners-staging");
  assert.equal(written.FIREBASE_EMAIL_LINK_DOMAIN, "");
  assert.equal(written.RESEND_API_KEY, "");
  assert.equal(written.RESEND_FROM, "");
  assert.equal(written.RESEND_REPLY_TO, "");
  assert.equal(written.RESEND_ASSET_BASE_URL, "");
  assert.equal(written.INSTAGRAM_USER_ID, "");
  assert.equal(written.INSTAGRAM_GRAPH_API_VERSION, "");
  assert.equal(written.INSTAGRAM_MEDIA_BASE_URL, "");
  assert.equal(written.CAMPAIGN_MEDIA_BUCKET, "");
  assert.equal(written.VEO_LOCATION, "");
  assert.equal(written.VEO_MODEL_ID, "");
});
