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

  assert.match(joinTemplate, /data-database-submit="\/api\/submit-join"/);
  assert.doesNotMatch(joinTemplate, /data-identity-signup-on-submit/);
  assert.doesNotMatch(joinTemplate, /name="ownership"/);
  assert.match(joinTemplate, /value="current-owner-one"/);
  assert.match(joinTemplate, /value="prospective-buyer"/);
  assert.match(joinTemplate, /data-enable-when-checked="consent-contact consent-not-legal"/);
  assert.match(joinTemplate, /data-enable-when-checked="consent-contact consent-not-legal"\s+disabled/);
  assert.match(joinTemplate, /Once you confirm the link, you can\s+sign in and add vehicle details/);
  assert.doesNotMatch(joinTemplate, /Vehicle evidence storage is still separate and not yet active/);

  assert.match(identityJs, /magicLinkSent/);
  assert.match(identityJs, /data-magic-link-form/);
  assert.match(identityJs, /window\.localStorage\.setItem\('ipaceEmailForSignIn', email\)/);
  assert.match(multistepJs, /data-enable-when-checked/);
  assert.match(multistepJs, /checkedRequirementsMet/);
  assert.match(multistepJs, /conditionalSubmitControlsMet/);
  assert.match(multistepJs, /focusFirstMissingCheckedRequirement/);
});
