'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

function read(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), 'utf8');
}

test('site UI uses Firebase passwordless magic-link forms', function () {
  var files = [
    'src/member/account.njk',
    'src/member/dashboard.njk',
    'src/member/submit-vehicle-data.njk',
    'src/admin/review-queue.njk',
    'src/assets/js/identity.js',
  ];

  files.forEach(function (file) {
    var source = read(file);
    assert.doesNotMatch(source, /data-identity-open/, file);
    assert.doesNotMatch(source, /identity\.open\(/, file);
  });

  var loginGate = read('src/_includes/partials/auth-login-gate.njk');
  assert.match(loginGate, /macro authLoginGate/);
  assert.match(loginGate, /data-magic-link-form/);
  assert.match(read('src/member/account.njk'), /authLoginGate\("account-magic-email"/);
  assert.match(read('src/member/dashboard.njk'), /authLoginGate\("dashboard-magic-email"/);
  assert.match(read('src/member/submit-vehicle-data.njk'), /authLoginGate\("vehicle-magic-email"/);
  assert.match(read('src/admin/review-queue.njk'), /authLoginGate\("admin-magic-email"/);
  assert.match(read('src/assets/js/identity.js'), /\/api\/send-magic-link/);
  assert.match(read('src/assets/js/identity.js'), /If this email address is registered/);
  assert.doesNotMatch(read('src/assets/js/identity.js'), /Check your email for a secure sign-in link/);
});

test('Join completion does not offer vehicle submission until signed in', function () {
  var join = read('src/join.njk') + read('src/_includes/partials/join-result.njk');
  var css = read('src/assets/css/site.css');

  assert.match(join, /data-registration-link-sent/);
  assert.match(join, /We are only asking owners to register now/);
  assert.match(join, /any later request for vehicle details will\s+be explained separately/);
  assert.doesNotMatch(join, /Add your first vehicle/);
  assert.doesNotMatch(join, /href="\/account\/" class="btn/);
  assert.match(css, /\[hidden\]\s*\{\s*display:\s*none !important;/);
});

test('Join completion separates saved state from magic-link delivery', function () {
  var join = read('src/join.njk') + read('src/_includes/partials/join-result.njk');

  assert.match(join, /Your membership details have been saved/);
  assert.match(join, /We asked our sign-in provider to email/);
  assert.match(join, /Delivery can be delayed or filtered/);
  assert.match(join, /Your join details were saved, but we couldn't send your sign-in link/);
  assert.doesNotMatch(join, /We've sent an email/);
  assert.doesNotMatch(join, /We're sending a sign-in link/);
});

test('Join completion uses a compact result card and removes pre-submit guidance', function () {
  var result = read('src/_includes/partials/join-result.njk');
  var css = read('src/assets/css/site.css');

  assert.match(result, /class="submit-result join-result"/);
  assert.match(result, /class="join-result__next-step"/);
  assert.doesNotMatch(result, /✅|📧/);
  assert.match(css, /\.join-result\s*\{[\s\S]*width: min\(100%, 44rem\)/);
  assert.match(css, /\.join-result strong\s*\{\s*display: inline;/);
  assert.match(css, /\.form-workspace\.is-submitted \.form-workspace__aside\s*\{\s*display: none;/);
});

test('member data fetches include Identity bearer tokens', function () {
  var memberAuth = read('src/assets/js/member-auth.js');

  assert.match(memberAuth, /function fetchWithIdentity/);
  assert.match(memberAuth, /headers\.Authorization = 'Bearer ' \+ token/);
  assert.match(memberAuth, /fetchWithIdentity\('\/api\/member-data'\)/);
  assert.match(memberAuth, /fetchWithIdentity\('\/api\/admin-data'\)/);
  assert.match(read('src/assets/js/identity.js'), /identity:ready/);
  assert.match(memberAuth, /addEventListener\('identity:ready'/);
  assert.doesNotMatch(read('src/assets/js/identity.js'), /window\.location\.reload/);
});

test('account preferences render from saved member data', function () {
  var account = read('src/member/account.njk');
  var memberAuth = read('src/assets/js/member-auth.js');
  var css = read('src/assets/css/site.css');

  assert.match(account, /data-preferences-container/);
  assert.doesNotMatch(account, /Notification and data use preferences will be manageable here in a future release/);
  assert.match(memberAuth, /function populatePreferences/);
  assert.match(memberAuth, /Group contact/);
  assert.match(memberAuth, /Anonymised aggregate analysis/);
  assert.match(memberAuth, /Participation acknowledgement/);
  assert.match(memberAuth, /Preference editing will be added with an audited account update flow/);
  assert.match(css, /\.preference-list/);
});

test('protected pages do not show login gates before auth verification completes', function () {
  var loginGate = read('src/_includes/partials/auth-login-gate.njk');

  [
    'src/member/dashboard.njk',
    'src/member/account.njk',
    'src/member/submit-vehicle-data.njk',
    'src/admin/review-queue.njk',
  ].forEach(function (file) {
    var source = read(file);
    assert.match(source, /import authLoginGate/, file);
    assert.match(source, /authLoginGate\(/, file);
  });

  assert.match(loginGate, /data-auth-pending/);
  assert.match(loginGate, /data-auth-login-gate hidden/);

  var memberAuth = read('src/assets/js/member-auth.js');
  var identity = read('src/assets/js/identity.js');

  assert.match(identity, /window\.ipaceIdentityReady = !config/);
  assert.match(identity, /window\.ipaceIdentityReady = true/);
  assert.match(memberAuth, /document\.addEventListener\('identity:ready', initSoon\)/);
  assert.match(memberAuth, /window\.ipaceIdentityReady/);
  assert.match(memberAuth, /setTimeout\(function \(\) \{/);
  assert.match(memberAuth, /document\.addEventListener\('identity:logout', initSoon\)/);
  assert.match(memberAuth, /var authRunId = 0/);
  assert.match(memberAuth, /if \(runId !== authRunId\) return/);
  assert.match(identity, /clearAuthQuery/);
  assert.match(identity, /mode=signIn\|oobCode=\|apiKey=/);
  assert.match(identity, /pendingEmailLinkUrl/);
  assert.match(identity, /completePendingEmailLink/);
  assert.match(identity, /Enter the email address that received this link to finish signing in/);
  assert.match(identity, /auth\.signInWithEmailLink\(email, pendingEmailLinkUrl\)/);
  assert.doesNotMatch(identity, /window\.prompt/);
});

test('homepage vehicle CTAs switch between guest and signed-in states', function () {
  var home = read('src/index.njk');

  assert.match(home, /data-requires-guest[\s\S]*Join to Submit Vehicle Data/);
  assert.match(home, /href="\/member\/submit-vehicle-data\/"[\s\S]*data-requires-auth/);
});

test('multi-step forms do not scroll on every step unless explicitly opted in', function () {
  var multistep = read('src/assets/js/multistep-form.js');

  assert.match(multistep, /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/join.njk'), /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/member/submit-vehicle-data.njk'), /data-scroll-on-step-change/);
});
