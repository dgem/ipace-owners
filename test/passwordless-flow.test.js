'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

function read(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), 'utf8');
}

test('site UI uses passwordless magic-link forms instead of Netlify modal triggers', function () {
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
  assert.match(read('src/assets/js/identity.js'), /\/\.netlify\/functions\/send-magic-link/);
});

test('Join completion does not offer vehicle submission until signed in', function () {
  var join = read('src/join.njk');
  var css = read('src/assets/css/site.css');

  assert.match(join, /data-registration-link-sent/);
  assert.match(join, /You can add vehicle data after opening the sign-in link/);
  assert.match(join, /data-registration-signed-in hidden[\s\S]*Add your first vehicle/);
  assert.match(css, /\[hidden\]\s*\{\s*display:\s*none !important;/);
});

test('member data fetches include Identity bearer tokens', function () {
  var memberAuth = read('src/assets/js/member-auth.js');

  assert.match(memberAuth, /function fetchWithIdentity/);
  assert.match(memberAuth, /Authorization: 'Bearer ' \+ token/);
  assert.match(memberAuth, /fetchWithIdentity\('\/\.netlify\/functions\/member-data'\)/);
  assert.match(memberAuth, /fetchWithIdentity\('\/\.netlify\/functions\/admin-data'\)/);
  assert.match(read('src/assets/js/identity.js'), /identity:ready/);
  assert.match(memberAuth, /addEventListener\('identity:ready'/);
  assert.doesNotMatch(read('src/assets/js/identity.js'), /window\.location\.reload/);
});

test('multi-step forms do not scroll on every step unless explicitly opted in', function () {
  var multistep = read('src/assets/js/multistep-form.js');

  assert.match(multistep, /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/join.njk'), /data-scroll-on-step-change/);
  assert.doesNotMatch(read('src/submit-vehicle-data.njk'), /data-scroll-on-step-change/);
});
