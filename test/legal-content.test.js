'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');

test('privacy and participation pages describe the live service rather than placeholders', function () {
  var privacy = fs.readFileSync(path.join(root, 'src/privacy.md'), 'utf8');
  var terms = fs.readFileSync(path.join(root, 'src/terms.md'), 'utf8');

  assert.doesNotMatch(privacy, /placeholder privacy|placeholder document/i);
  assert.doesNotMatch(terms, /placeholder participation|placeholder document/i);
  assert.match(privacy, /controller for personal data/i);
  assert.match(privacy, /lawful bases/i);
  assert.match(privacy, /How long we keep data/i);
  assert.match(privacy, /Google Cloud and Firebase/);
  assert.match(privacy, /Resend/);
  assert.match(privacy, /right[s]? to:[\s\S]*Object to processing/i);
  assert.match(privacy, /complain to the[\s\S]*Information Commissioner's Office/i);
  assert.match(terms, /personal exports/i);
});
