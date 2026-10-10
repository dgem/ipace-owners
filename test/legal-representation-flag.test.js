'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');

test('legal representation interest remains disabled until deliberately launched', function () {
  var site = JSON.parse(fs.readFileSync(path.join(root, 'src/_data/site.json'), 'utf8'));
  var account = fs.readFileSync(path.join(root, 'src/member/account.njk'), 'utf8');
  var profile = fs.readFileSync(path.join(root, 'functions/firebase-go/member_profile.go'), 'utf8');

  assert.equal(site.features.legalRepresentation, false);
  assert.match(account, /{% if site\.features\.legalRepresentation %}/);
  assert.match(profile, /LEGAL_REPRESENTATION_ENABLED/);
  assert.match(profile, /preserveRepresentationWhenDisabled/);
});
