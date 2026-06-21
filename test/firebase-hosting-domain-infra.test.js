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
  assert.match(moduleMain, /wait_dns_verification\s*=\s*var\.firebase_hosting_wait_for_dns_verification/);
  assert.match(moduleMain, /deletion_policy\s*=\s*"ABANDON"/);
  assert.match(moduleOutputs, /required_dns_updates/);
  assert.match(moduleOutputs, /ownership_state/);
  assert.match(moduleOutputs, /certificate_state/);
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
