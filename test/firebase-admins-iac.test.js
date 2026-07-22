const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');
const scriptPath = resolve(repoRoot, 'scripts/reconcile-firebase-admins.mjs');

test('normalises and validates the administrator email map', async function () {
  const { parseAdminUsers } = await import(scriptPath);
  assert.deepEqual(
    parseAdminUsers('{"dan":" Dan@Kanzi.co.uk "}'),
    { dan: 'dan@kanzi.co.uk' },
  );
  assert.throws(() => parseAdminUsers('{}'), /empty desired set/);
  assert.throws(() => parseAdminUsers('{"dan":"not-an-email"}'), /invalid email/);
  assert.throws(
    () => parseAdminUsers('{"one":"dan@kanzi.co.uk","two":"DAN@KANZI.CO.UK"}'),
    /must be unique/,
  );
});

test('grants and revokes only admin access while preserving other claims', async function () {
  const { buildAdminUpdates } = await import(scriptPath);
  const users = [
    {
      localId: 'dan-production-uid',
      email: 'dan@kanzi.co.uk',
      customAttributes: JSON.stringify({ accessLevel: 10, nested: { retained: true } }),
    },
    {
      localId: 'former-admin-uid',
      email: 'former@example.org',
      customAttributes: JSON.stringify({ admin: true, roles: ['admin', 'reviewer'], retained: 'yes' }),
    },
    {
      localId: 'member-uid',
      email: 'member@example.org',
      customAttributes: JSON.stringify({ retained: true }),
    },
  ];
  const updates = buildAdminUpdates(users, { dan: 'dan@kanzi.co.uk' });

  assert.equal(updates.length, 2);
  const grant = updates.find((update) => update.uid === 'dan-production-uid');
  const revocation = updates.find((update) => update.uid === 'former-admin-uid');
  assert.deepEqual(JSON.parse(grant.customAttributes), {
    accessLevel: 10,
    nested: { retained: true },
    admin: true,
  });
  assert.deepEqual(JSON.parse(revocation.customAttributes), { roles: ['reviewer'], retained: 'yes' });
  assert.equal(grant.grant, true);
  assert.equal(revocation.grant, false);
});

test('requires every configured administrator account to exist', async function () {
  const { buildAdminUpdates } = await import(scriptPath);
  assert.throws(
    () => buildAdminUpdates([], { dan: 'dan@kanzi.co.uk' }),
    /Configured Firebase administrators do not exist: dan/,
  );
});

test('OpenTofu always includes the required owner and reconciles through Identity Platform', function () {
  const moduleMain = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/main.tf'), 'utf8');
  const moduleVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/variables.tf'), 'utf8');
  const envVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/env/variables.tf'), 'utf8');
  const script = readFileSync(scriptPath, 'utf8');

  assert.match(moduleMain, /firebase_admin_users\s*=\s*merge\(var\.firebase_admin_users/);
  assert.match(moduleMain, /dan\s*=\s*"dan@kanzi\.co\.uk"/);
  assert.match(moduleMain, /resource "terraform_data" "firebase_admins"/);
  assert.match(moduleMain, /FIREBASE_ADMIN_USERS_JSON\s*=\s*jsonencode\(local\.firebase_admin_users\)/);
  assert.match(moduleVariables, /variable "manage_firebase_admins"[\s\S]*default\s*=\s*true/);
  assert.match(envVariables, /variable "manage_firebase_admins"[\s\S]*default\s*=\s*true/);
  assert.match(script, /accounts:batchGet/);
  assert.match(script, /accounts:update/);
  assert.match(script, /delete next\.admin/);
  assert.doesNotMatch(script, /console\.log\([^\n]*(?:email|uid)/i);
});
