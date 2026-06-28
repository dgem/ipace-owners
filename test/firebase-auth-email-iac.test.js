const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');
const scriptPath = resolve(repoRoot, 'scripts/configure-firebase-auth-email.mjs');

test('builds branded Firebase Auth email configuration with a custom action domain', async function () {
  const {
    buildEmailConfig,
    emailConfigUpdateMask,
    emailDomainVerificationEndpoint,
    normalizeDomain,
  } = await import(scriptPath);
  const templates = {
    resetPasswordTemplate: '<a href="%LINK%">Reset</a>',
    verifyEmailTemplate: '<a href="%LINK%">Verify</a>',
    changeEmailTemplate: '<a href="%LINK%">Restore %NEW_EMAIL%</a>',
    revertSecondFactorAdditionTemplate: '<a href="%LINK%">Remove</a>',
  };
  const config = buildEmailConfig({
    actionDomain: normalizeDomain('IPACE-OWNERS.ORG', 'action domain'),
    emailDomain: 'ipace-owners.org',
    senderLocalPart: 'members',
    senderDisplayName: 'I-PACE Owners Advocacy Group',
    replyTo: 'contact@ipace-owners.org',
  }, templates);

  assert.equal(config.notification.defaultLocale, 'en-GB');
  assert.equal(config.notification.sendEmail.callbackUri, 'https://ipace-owners.org/__/auth/action');
  assert.equal(config.notification.sendEmail.verifyEmailTemplate.senderDisplayName, 'I-PACE Owners Advocacy Group');
  assert.equal(config.notification.sendEmail.verifyEmailTemplate.bodyFormat, 'HTML');
  assert.equal(config.notification.sendEmail.dnsInfo.useCustomDomain, true);
  assert.match(emailConfigUpdateMask(true), /notification\.sendEmail\.dnsInfo\.useCustomDomain/);
  assert.equal(
    emailDomainVerificationEndpoint('https://identitytoolkit.googleapis.com/admin/v2/projects/example/config'),
    'https://identitytoolkit.googleapis.com/admin/v2/projects/example/domain:verify',
  );
  assert.throws(() => normalizeDomain('https://ipace-owners.org/account/', 'action domain'));
});

test('stores professional Firebase Auth templates and manages them through infrastructure', function () {
  const moduleMain = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/main.tf'), 'utf8');
  const moduleVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/variables.tf'), 'utf8');
  const makefile = readFileSync(resolve(repoRoot, 'Makefile'), 'utf8');
  const script = readFileSync(scriptPath, 'utf8');

  assert.match(moduleMain, /resource "terraform_data" "firebase_auth_email"/);
  assert.match(moduleMain, /configure-firebase-auth-email\.mjs/);
  assert.match(moduleVariables, /variable "firebase_auth_email_domain"/);
  assert.match(makefile, /infra-email-domain:/);
  assert.match(script, /\/domain:verify/);
  assert.match(script, /action: "VERIFY"/);
  assert.match(script, /action: "APPLY"/);

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
