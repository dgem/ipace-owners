const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');
const scriptPath = resolve(repoRoot, 'scripts/configure-firebase-auth-email.mjs');

test('builds supported passwordless email configuration without restricted fields', async function () {
  const {
    buildEmailConfig,
    emailConfigUpdateMask,
    emailConfigUpdates,
    emailDomainVerificationEndpoint,
    normalizeDomain,
  } = await import(scriptPath);
  const config = buildEmailConfig({
    emailDomain: 'ipace-owners.org',
    senderLocalPart: 'members',
    senderDisplayName: 'I-PACE Owners Advocacy Group',
    replyTo: 'contact@ipace-owners.org',
  });

  assert.equal(config.notification.defaultLocale, 'en-GB');
  assert.equal(config.notification.sendEmail.callbackUri, undefined);
  assert.equal(config.notification.sendEmail.verifyEmailTemplate, undefined);
  assert.equal(config.notification.sendEmail.dnsInfo, undefined);
  assert.doesNotMatch(emailConfigUpdateMask(), /notification\.sendEmail\.dnsInfo/);
  const updates = emailConfigUpdates(config);
  assert.equal(updates.length, 2);
  assert.equal(updates[0].name, 'default locale');
  assert.ok(updates.every((update) => !update.mask.includes('callbackUri')));
  assert.ok(updates.every((update) => !update.mask.includes('Template')));
  assert.ok(updates.every((update) => !update.mask.includes('dnsInfo')));
  assert.equal(
    emailDomainVerificationEndpoint('https://identitytoolkit.googleapis.com/admin/v2/projects/example/config'),
    'https://identitytoolkit.googleapis.com/admin/v2/projects/example/domain:verify',
  );
  assert.throws(() => normalizeDomain('https://ipace-owners.org/account/', 'action domain'));
});

test('stores future email designs and manages supported settings through infrastructure', function () {
  const moduleMain = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/main.tf'), 'utf8');
  const moduleVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/variables.tf'), 'utf8');
  const makefile = readFileSync(resolve(repoRoot, 'Makefile'), 'utf8');
  const script = readFileSync(scriptPath, 'utf8');

  assert.match(moduleMain, /resource "terraform_data" "firebase_auth_email"/);
  assert.match(moduleMain, /configure-firebase-auth-email\.mjs/);
  assert.match(moduleMain, /filesha256\(local\.firebase_auth_email_script\)/);
  assert.doesNotMatch(moduleMain, /FIREBASE_AUTH_EMAIL_TEMPLATE_DIR/);
  assert.match(moduleVariables, /variable "firebase_auth_email_domain"/);
  assert.match(makefile, /infra-email-domain:/);
  assert.match(script, /\/domain:verify/);
  assert.match(script, /action: "VERIFY"/);
  assert.match(script, /action: "APPLY"/);
  assert.doesNotMatch(script, /fields\.push\("notification\.sendEmail\.dnsInfo\.useCustomDomain"\)/);

  for (const filename of [
    'reset-password.html',
    'verify-email.html',
    'change-email.html',
    'revert-second-factor.html',
  ]) {
    const template = readFileSync(
      resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/templates/auth-email', filename),
      'utf8',
    );
    assert.match(template, /I-PACE Owners Advocacy Group/);
    assert.match(template, /%LINK%/);
  }
});
