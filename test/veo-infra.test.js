const assert = require("node:assert/strict");
const { readFileSync } = require("node:fs");
const { resolve } = require("node:path");
const { test } = require("node:test");

function read(relativePath) {
  return readFileSync(resolve(__dirname, "..", relativePath), "utf8");
}

test("OpenTofu enables Vertex AI and provisions least-privilege Veo media storage", () => {
  const main = read("infra/opentofu/modules/ipace-owners/main.tf");
  const variables = read("infra/opentofu/modules/ipace-owners/variables.tf");

  assert.match(main, /"aiplatform\.googleapis\.com"/);
  assert.match(main, /resource "google_storage_bucket" "campaign_media"/);
  assert.match(main, /public_access_prevention\s+=\s+"enforced"/);
  assert.match(main, /uniform_bucket_level_access\s+=\s+true/);
  assert.match(main, /age\s+=\s+var\.campaign_media_work_retention_days/);
  assert.match(main, /matches_prefix\s+=\s+\["work\/"\]/);
  assert.match(main, /versioning\s+\{\s+enabled\s+=\s+true/s);
  assert.match(main, /role\s+=\s+"roles\/aiplatform\.user"/);
  assert.match(main, /resource "google_project_service_identity" "vertex_ai"/);
  assert.match(main, /service\s+=\s+"aiplatform\.googleapis\.com"/);
  assert.match(main, /resource "google_project_iam_member" "vertex_ai_service_agent"/);
  assert.match(main, /role\s+=\s+"roles\/aiplatform\.serviceAgent"/);
  assert.match(main, /resource "google_storage_bucket_iam_member" "runtime_campaign_media"/);
  assert.match(main, /resource "google_storage_bucket_iam_member" "vertex_ai_campaign_media"/);
  assert.match(main, /google_project_service_identity\.vertex_ai\.email/);
  assert.match(variables, /default\s+=\s+"veo-3\.1-generate-001"/);
  assert.match(variables, /default\s+=\s+"us-central1"/);
  assert.match(variables, /var\.veo_location == "us-central1"/);
});

test("Veo configuration flows from OpenTofu through both Function deployment environments", () => {
  const githubActions = read("infra/opentofu/modules/ipace-owners/github-actions.tf");
  const staging = read(".github/workflows/gcp-firebase-staging.yml");
  const production = read(".github/workflows/gcp-firebase-production.yml");

  for (const name of ["CAMPAIGN_MEDIA_BUCKET", "VEO_LOCATION", "VEO_MODEL_ID"]) {
    assert.match(githubActions, new RegExp(`${name}_\\$\\{local\\.github_actions_suffix\\}`));
    assert.match(staging, new RegExp(`${name}: \\$\\{\\{ vars\\.${name}_STAGING \\}\\}`));
    assert.match(production, new RegExp(`${name}: \\$\\{\\{ vars\\.${name}_PRODUCTION \\}\\}`));
  }
});

test("Meta OAuth token storage is declared without putting token bytes in OpenTofu state", () => {
  const main = read("infra/opentofu/modules/ipace-owners/main.tf");
  const githubActions = read("infra/opentofu/modules/ipace-owners/github-actions.tf");
  const makefile = read("Makefile");

  assert.match(main, /resource "google_secret_manager_secret" "instagram_access_token"/);
  assert.doesNotMatch(main, /resource "google_secret_manager_secret_version" "instagram_access_token"/);
  assert.match(main, /resource "google_secret_manager_secret_iam_member" "runtime_instagram_access_token"/);
  assert.match(githubActions, /var\.instagram_publishing_enabled \? \{/);
  assert.doesNotMatch(githubActions, /INSTAGRAM_ACCESS_TOKEN_SECRET_/);
  assert.doesNotMatch(githubActions, /INSTAGRAM_ACCESS_TOKEN_/);
  assert.match(makefile, /INSTAGRAM_ACCESS_TOKEN=instagram-access-token:latest/);
  assert.match(makefile, /VIN_PEPPER=vin-pepper:latest/);
  assert.match(makefile, /RESEND_API_KEY=resend-api-key:latest/);
  assert.match(makefile, /IDEAL_POSTCODES_API_KEY=ideal-postcodes-api-key:latest/);
});

test("application secret containers are protected from accidental removal", () => {
  const main = read("infra/opentofu/modules/ipace-owners/main.tf");

  for (const name of ["vin_pepper", "resend_api_key", "ideal_postcodes_api_key", "instagram_access_token"]) {
    assert.match(main, new RegExp(`resource "google_secret_manager_secret" "${name}"[\\s\\S]*?deletion_policy\\s+=\\s+"ABANDON"[\\s\\S]*?deletion_protection\\s+=\\s+true`));
  }
});

test("local secret bootstrap is automatic, idempotent, and does not use Terraform inputs", () => {
  const main = read("infra/opentofu/modules/ipace-owners/main.tf");
  const variables = read("infra/opentofu/modules/ipace-owners/variables.tf");
  const bootstrap = read("functions/firebase-go/cmd/secret-version-bootstrap/main.go");

  assert.match(main, /resource "terraform_data" "local_secret_versions"[\s\S]*?triggers_replace\s+=\s+\[timestamp\(\)\]/);
  assert.doesNotMatch(main, /SECRET_VERSION_ROTATIONS|secret_version_rotation|manage_local_secret_versions/);
  assert.doesNotMatch(variables, /secret_version_rotation|manage_local_secret_versions/);
  assert.doesNotMatch(bootstrap, /SECRET_VERSION_ROTATIONS/);
  assert.match(bootstrap, /not set locally; skipping/);
  for (const name of ["VIN_PEPPER", "RESEND_API_KEY", "IDEAL_POSTCODES_API_KEY", "INSTAGRAM_ACCESS_TOKEN"]) {
    assert.match(bootstrap, new RegExp(`"${name}"`));
  }
});
