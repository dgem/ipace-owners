'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('Join form submits only to submit-join from the browser', function () {
  var joinTemplate = fs.readFileSync(path.join(repoRoot, 'src/join.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');

  assert.match(joinTemplate, /data-database-submit="\/\.netlify\/functions\/submit-join"/);
  assert.doesNotMatch(joinTemplate, /data-identity-signup-on-submit/);
  assert.doesNotMatch(joinTemplate, /data-netlify=/);
  assert.doesNotMatch(joinTemplate, /name="ownership"/);
  assert.match(joinTemplate, /value="current-owner-one"/);
  assert.match(joinTemplate, /value="prospective-buyer"/);

  assert.doesNotMatch(identityJs, /fetch\(['"]\/\.netlify\/functions\/send-magic-link/);
  assert.match(identityJs, /magicLinkSent/);
});
