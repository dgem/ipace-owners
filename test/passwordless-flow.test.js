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
    'src/account.njk',
    'src/member/dashboard.njk',
    'src/submit-vehicle-data.njk',
    'src/admin/review-queue.njk',
    'src/assets/js/identity.js',
  ];

  files.forEach(function (file) {
    var source = read(file);
    assert.doesNotMatch(source, /data-identity-open/, file);
    assert.doesNotMatch(source, /identity\.open\(/, file);
  });

  assert.match(read('src/account.njk'), /data-magic-link-form/);
  assert.match(read('src/member/dashboard.njk'), /data-magic-link-form/);
  assert.match(read('src/submit-vehicle-data.njk'), /data-magic-link-form/);
  assert.match(read('src/admin/review-queue.njk'), /data-magic-link-form/);
  assert.match(read('src/assets/js/identity.js'), /\/api\/send-magic-link/);
  assert.match(read('src/assets/js/identity.js'), /If this email address is registered/);
  assert.doesNotMatch(read('src/assets/js/identity.js'), /Check your email for a secure sign-in link/);
});

test('Join completion does not offer vehicle submission until signed in', function () {
  var join = read('src/join.njk');
  var css = read('src/assets/css/site.css');

  assert.match(join, /data-registration-link-sent/);
  assert.match(join, /You can add vehicle data after\s+opening the sign-in link/);
  assert.match(join, /data-registration-signed-in hidden[\s\S]*Add your first vehicle/);
  assert.match(css, /\[hidden\]\s*\{\s*display:\s*none !important;/);
});

test('Join completion separates saved state from magic-link delivery', function () {
  var join = read('src/join.njk');

  assert.match(join, /Your join details have been saved/);
  assert.match(join, /sign-in provider accepted the request/);
  assert.match(join, /Delivery can be delayed or filtered/);
  assert.match(join, /Your join details were saved, but we couldn't send your sign-in link/);
  assert.doesNotMatch(join, /We've sent an email/);
  assert.doesNotMatch(join, /We're sending a sign-in link/);
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

test('protected pages do not show login gates before auth verification completes', function () {
  [
    'src/member/dashboard.njk',
    'src/account.njk',
    'src/submit-vehicle-data.njk',
    'src/admin/review-queue.njk',
  ].forEach(function (file) {
    var source = read(file);
    assert.match(source, /data-auth-pending/, file);
    assert.match(source, /data-auth-login-gate hidden/, file);
  });

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
});

test('homepage vehicle CTAs switch between guest and signed-in states', function () {
  var home = read('src/index.njk');

  assert.match(home, /data-requires-guest[\s\S]*Join to Submit Vehicle Data/);
  assert.match(home, /href="\/submit-vehicle-data\/"[\s\S]*data-requires-auth/);
});

test('multi-step forms do not scroll on every step unless explicitly opted in', function () {
  var multistep = read('src/assets/js/multistep-form.js');

  assert.match(multistep, /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/join.njk'), /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/submit-vehicle-data.njk'), /data-scroll-on-step-change/);
});
