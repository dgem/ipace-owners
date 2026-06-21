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
      FIREBASE_WEB_API_KEY: "api-key",
      VIN_PEPPER: "pepper",
      SNAPSHOT_BUCKET: "snapshots",
      ALLOWED_ORIGINS: "https://stage.ipace-owners.org,http://localhost:8080,http://localhost:5000",
      FIREBASE_EMAIL_CONTINUE_URL: "https://stage.ipace-owners.org/account/",
      FIREBASE_EMAIL_LINK_DOMAIN: "stage.ipace-owners.org",
    },
  });

  const written = JSON.parse(readFileSync(join(cwd, "functions-env.json"), "utf8"));

  assert.deepEqual(written, {
    FIREBASE_PROJECT_ID: "ipace-owners-staging",
    FIREBASE_WEB_API_KEY: "api-key",
    VIN_PEPPER: "pepper",
    SNAPSHOT_BUCKET: "snapshots",
    ALLOWED_ORIGINS: "https://stage.ipace-owners.org,http://localhost:8080,http://localhost:5000",
    FIREBASE_EMAIL_CONTINUE_URL: "https://stage.ipace-owners.org/account/",
    FIREBASE_EMAIL_LINK_DOMAIN: "stage.ipace-owners.org",
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
