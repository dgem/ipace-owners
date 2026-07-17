const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');

function read(path) {
  return readFileSync(resolve(repoRoot, path), 'utf8');
}

test('OpenTofu manages Firebase Hosting custom domains and reports DNS state', function () {
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');
  const moduleOutputs = read('infra/opentofu/modules/ipace-owners/outputs.tf');

  assert.match(moduleMain, /google_firebase_hosting_custom_domain/);
  assert.match(moduleMain, /wait_dns_verification\s*=\s*false/);
  assert.match(moduleMain, /deletion_policy\s*=\s*"PREVENT"/);
  assert.match(moduleOutputs, /required_dns_updates/);
  assert.match(moduleOutputs, /certificate\.verification/);
  assert.match(moduleOutputs, /verification\.dns/);
  assert.match(moduleOutputs, /ownership_state/);
  assert.match(moduleOutputs, /certificate_state/);
});

test('OpenTofu can manage Resend domains and report DNS state', function () {
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');
  const moduleOutputs = read('infra/opentofu/modules/ipace-owners/outputs.tf');
  const envOutputs = read('infra/opentofu/env/outputs.tf');

  assert.match(moduleMain, /resource "resend_domain" "auth_email"/);
  assert.match(moduleMain, /count\s*=\s*var\.manage_resend_domain && var\.resend_domain != "" \? 1 : 0/);
  assert.match(moduleOutputs, /output "resend_email_domain"/);
  assert.match(moduleOutputs, /dns_records/);
  assert.match(envOutputs, /output "resend_email_domain"/);
});

test('environment examples define staging and canonical production domains', function () {
  const staging = read('infra/opentofu/env/staging.tfvars.example');
  const production = read('infra/opentofu/env/production.tfvars.example');

  assert.match(staging, /"stage\.ipace-owners\.org"/);
  assert.match(production, /"ipace-owners\.org"\s*=\s*\{\}/);
  assert.match(production, /"www\.ipace-owners\.org"\s*=\s*\{ redirect_target = "ipace-owners\.org" \}/);
});

test('Makefile exposes Firebase Hosting DNS records for an explicit environment', function () {
  const makefile = read('Makefile');

  assert.match(makefile, /infra-dns-records:.*##/);
  assert.match(makefile, /\$\(INFRA_ENV_SCRIPT\) dns/);
});

test('Makefile exposes Resend DNS records for an explicit environment', function () {
  const makefile = read('Makefile');
  const infraScript = read('scripts/infra-env.sh');

  assert.match(makefile, /infra-resend-dns-records:.*##/);
  assert.match(makefile, /\$\(INFRA_ENV_SCRIPT\) resend-dns/);
  assert.match(infraScript, /resend-dns\)/);
  assert.match(infraScript, /output resend_email_domain/);
});

test('OpenTofu and deployment workflows select the named Firestore database', function () {
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');
  const moduleVariables = read('infra/opentofu/modules/ipace-owners/github-actions.tf');
  const productionWorkflow = read('.github/workflows/gcp-firebase-production.yml');
  const stagingWorkflow = read('.github/workflows/gcp-firebase-staging.yml');

  assert.match(moduleMain, /resource "google_firestore_database" "default"[\s\S]*name\s*=\s*var\.project_id/);
  assert.match(moduleMain, /resource "google_firestore_database" "default"[\s\S]*deletion_policy\s*=\s*local\.production_data_protection \? "PREVENT" : "ABANDON"/);
  assert.match(moduleVariables, /FIRESTORE_DATABASE_ID_/);
  assert.match(productionWorkflow, /FIRESTORE_DATABASE_ID_PRODUCTION/);
  assert.match(stagingWorkflow, /FIRESTORE_DATABASE_ID_STAGING/);
});

test('production Firestore has delete protection, PITR, and scheduled backups', function () {
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');
  const moduleOutputs = read('infra/opentofu/modules/ipace-owners/outputs.tf');
  const envOutputs = read('infra/opentofu/env/outputs.tf');
  const productionExample = read('infra/opentofu/env/production.tfvars.example');
  const stagingExample = read('infra/opentofu/env/staging.tfvars.example');

  assert.match(moduleMain, /production_data_protection\s*=\s*var\.environment == "production"/);
  assert.match(moduleMain, /point_in_time_recovery_enablement\s*=\s*local\.production_data_protection \? "POINT_IN_TIME_RECOVERY_ENABLED" : null/);
  assert.match(moduleMain, /delete_protection_state\s*=\s*local\.production_data_protection \? "DELETE_PROTECTION_ENABLED" : null/);
  assert.match(moduleMain, /resource "google_firestore_backup_schedule" "default"[\s\S]*count\s*=\s*local\.production_data_protection \? 1 : 0/);
  assert.match(moduleMain, /retention\s*=\s*local\.firestore_backup_retention/);
  assert.match(moduleMain, /daily_recurrence\s*\{\}/);
  assert.match(moduleMain, /resource "google_firestore_backup_schedule" "default"[\s\S]*deletion_policy\s*=\s*"PREVENT"/);
  assert.match(moduleMain, /firestore_backup_retention\s*=\s*"8467200s"/);
  assert.match(moduleOutputs, /encrypted_at_rest\s*=\s*"GOOGLE_MANAGED"/);
  assert.match(moduleOutputs, /backup_schedule_enabled/);
  assert.match(envOutputs, /output "firestore_data_protection"/);
  assert.doesNotMatch(stagingExample, /firestore_backup|delete_protection|point_in_time_recovery/);
  assert.doesNotMatch(productionExample, /firestore_backup|delete_protection|point_in_time_recovery/);
});

test('the GitHub deployer can update only Firebase Auth preview-domain configuration', function () {
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');

  assert.match(moduleMain, /resource "google_project_iam_custom_role" "github_firebase_auth_config"/);
  assert.match(moduleMain, /"firebaseauth\.configs\.get"/);
  assert.match(moduleMain, /"firebaseauth\.configs\.update"/);
  assert.doesNotMatch(moduleMain, /github_deployer_roles[\s\S]*"roles\/identitytoolkit\.admin"/);
});
