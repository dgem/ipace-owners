'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('Join form submits only to submit-join from the browser', function () {
  var joinTemplate = fs.readFileSync(path.join(repoRoot, 'src/join.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');
  var multistepJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/multistep-form.js'), 'utf8');

  assert.equal((joinTemplate.match(/data-database-submit="\/api\/submit-join"/g) || []).length, 2);
  assert.doesNotMatch(joinTemplate, /data-identity-signup-on-submit/);
  assert.doesNotMatch(joinTemplate, /name="ownership"/);
  assert.match(joinTemplate, /value="current-owner-one"/);
  assert.match(joinTemplate, /value="prospective-buyer"/);
  assert.match(joinTemplate, /data-enable-when-checked="consent-contact consent-not-legal"/);
  assert.match(joinTemplate, /data-enable-when-checked="consent-contact consent-not-legal"\s+disabled/);
  assert.match(
    joinTemplate,
    /Once you confirm your email, you can add vehicle details, battery readings,\s+and service or fault history/
  );
  assert.doesNotMatch(joinTemplate, /Vehicle evidence storage is still separate and not yet active/);
  assert.ok(
    joinTemplate.indexOf('name="join"') < joinTemplate.indexOf('Membership details are saved'),
    'Join informational callout should appear below the form'
  );

  assert.match(identityJs, /magicLinkSent/);
  assert.match(identityJs, /data-magic-link-form/);
  assert.match(identityJs, /window\.localStorage\.setItem\('ipaceEmailForSignIn', email\)/);
  assert.match(multistepJs, /data-enable-when-checked/);
  assert.match(multistepJs, /checkedRequirementsMet/);
  assert.match(multistepJs, /checkboxControlsByName/);
  assert.doesNotMatch(multistepJs, /control\.name === name && control\.type === 'checkbox' && control\.checked/);
  assert.match(multistepJs, /conditionalSubmitControlsMet/);
  assert.match(multistepJs, /focusFirstMissingCheckedRequirement/);
  assert.match(multistepJs, /workspace\.classList\.add\('is-submitted'\)/);
});

test('launch mode uses a one-step minimum-data Join form and retains the extended form', function () {
  var joinTemplate = fs.readFileSync(path.join(repoRoot, 'src/join.njk'), 'utf8');
  var launchStart = joinTemplate.indexOf('data-site-mode-only="launch"');
  var fullStart = joinTemplate.indexOf('data-site-mode-only="full"');
  var launchForm = joinTemplate.slice(launchStart, fullStart);
  var fullForm = joinTemplate.slice(fullStart);

  assert.ok(launchStart > -1 && fullStart > launchStart);
  assert.equal((launchForm.match(/data-step=/g) || []).length, 1);
  ['name="name"', 'name="email"', 'name="consent-contact"', 'name="consent-not-legal"', 'name="consent-data"'].forEach(function (field) {
    assert.match(launchForm, new RegExp(field));
  });
  ['name="country"', 'name="relationship"', 'name="skills"'].forEach(function (field) {
    assert.doesNotMatch(launchForm, new RegExp(field));
    assert.match(fullForm, new RegExp(field));
  });
  assert.match(launchForm, /anonymised aggregate form/);
  assert.match(launchForm, /Privacy Policy/);
  assert.match(launchForm, /Participation Statement/);
  assert.match(launchForm, /data-enable-when-checked="consent-contact consent-not-legal"/);
});
