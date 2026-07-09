'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');
var faq = fs.readFileSync(path.join(repoRoot, 'src', 'faq.njk'), 'utf8');
var about = fs.readFileSync(path.join(repoRoot, 'src', 'about.md'), 'utf8');

test('launch FAQ sets UK-led scope and keeps legal participation deliberate', function () {
  assert.match(faq, /Can non-UK owners register interest/);
  assert.match(faq, /this is a UK-led initiative/);
  assert.match(faq, /future legal\s+proposal would be explained separately/);
  assert.match(faq, /Can members still pursue their own complaint or claim/);
  assert.match(faq, /keep the collective\s+approach aligned/);
  assert.doesNotMatch(faq, /collect data from across the global I-PACE fleet/);
});

test('about page describes aligned advocacy rather than an open forum', function () {
  assert.match(about, /UK-led because UK owners\s+share the same market context/);
  assert.match(about, /not building an open comment board/);
  assert.match(about, /recording overseas interest separately/);
});
