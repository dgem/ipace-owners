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
    normalizeDisplayName,
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
  assert.equal(normalizeDisplayName(' I-PACE Owners '), 'I-PACE Owners');
  assert.throws(() => normalizeDisplayName('x'.repeat(81)));
});

test('stores future email designs and manages supported settings through infrastructure', function () {
  const moduleMain = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/main.tf'), 'utf8');
  const moduleVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/variables.tf'), 'utf8');
  const githubActions = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/github-actions.tf'), 'utf8');
  const envVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/env/variables.tf'), 'utf8');
  const stagingConfig = readFileSync(resolve(repoRoot, 'infra/opentofu/env/staging.tfvars.example'), 'utf8');
  const productionConfig = readFileSync(resolve(repoRoot, 'infra/opentofu/env/production.tfvars.example'), 'utf8');
  const makefile = readFileSync(resolve(repoRoot, 'Makefile'), 'utf8');
  const script = readFileSync(scriptPath, 'utf8');

  assert.match(moduleMain, /resource "terraform_data" "firebase_auth_email"/);
  assert.match(moduleMain, /configure-firebase-auth-email\.mjs/);
  assert.match(moduleMain, /filesha256\(local\.firebase_auth_email_script\)/);
  assert.match(moduleMain, /firebase_project_display_name/);
  assert.match(moduleMain, /FIREBASE_PROJECT_DISPLAY_NAME\s*=\s*local\.firebase_project_display_name/);
  assert.doesNotMatch(moduleMain, /FIREBASE_AUTH_EMAIL_TEMPLATE_DIR/);
  assert.match(moduleVariables, /variable "firebase_auth_email_domain"/);
  assert.match(moduleVariables, /variable "firebase_project_display_name"/);
  assert.match(envVariables, /variable "firebase_project_display_name"/);
  assert.match(githubActions, /FIREBASE_EMAIL_LINK_DOMAIN_\$\{local\.github_actions_suffix\}"\s*=\s*local\.firebase_auth_email_action_domain/);
  assert.doesNotMatch(githubActions, /FIREBASE_EMAIL_LINK_DOMAIN_\$\{local\.github_actions_suffix\}"\s*=\s*local\.email_continue_host/);
  assert.doesNotMatch(moduleVariables, /firebase_auth_email_reply_to/);
  assert.doesNotMatch(envVariables, /firebase_auth_email_sender_display_name/);
  assert.match(stagingConfig, /firebase_project_display_name\s*=\s*"I-PACE Owners Staging"/);
  assert.match(productionConfig, /firebase_project_display_name\s*=\s*"I-PACE Owners"/);
  assert.match(stagingConfig, /firebase_auth_email_domain\s*=\s*"auth\.stage\.ipace-owners\.org"/);
  assert.match(productionConfig, /firebase_auth_email_domain\s*=\s*"auth\.ipace-owners\.org"/);
  assert.match(moduleVariables, /variable "resend_from"/);
  assert.match(envVariables, /variable "resend_from"/);
  assert.match(githubActions, /RESEND_FROM_\$\{local\.github_actions_suffix\}/);
  assert.match(githubActions, /RESEND_REPLY_TO_\$\{local\.github_actions_suffix\}/);
  assert.match(githubActions, /RESEND_ASSET_BASE_URL_\$\{local\.github_actions_suffix\}/);
  assert.match(productionConfig, /resend_from\s*=\s*"I-PACE Owners <members@ipace-owners\.org>"/);
  assert.match(productionConfig, /resend_asset_base_url\s*=\s*"https:\/\/ipace-owners\.org"/);
  assert.match(makefile, /infra-email-domain:/);
  assert.match(script, /\/domain:verify/);
  assert.match(script, /action: "VERIFY"/);
  assert.match(script, /action: "APPLY"/);
  assert.match(script, /firebase\.googleapis\.com\/v1beta1\/projects/);
  assert.match(script, /updateMask=.*displayName/);
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
