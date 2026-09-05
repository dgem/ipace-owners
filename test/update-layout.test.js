'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');
var updatesDirectory = path.join(root, 'src', 'updates');

test('update posts use the dedicated editorial layout', function () {
  fs.readdirSync(updatesDirectory)
    .filter(function (file) { return file.endsWith('.md'); })
    .forEach(function (file) {
      var source = fs.readFileSync(path.join(updatesDirectory, file), 'utf8');
      assert.match(source, /layout: update\.njk/, file + ' should use update.njk');
    });
});

test('the update layout keeps article context and optional heroes together', function () {
  var layout = fs.readFileSync(path.join(root, 'src', '_includes', 'layouts', 'update.njk'), 'utf8');

  assert.match(layout, /date \| readableDate/);
  assert.match(layout, /I-PACE Owners' Advocacy Group/);
  assert.match(layout, /update-header__summary/);
  assert.match(layout, /update-article/);
  assert.match(layout, /{% if heroImage %}/);
});

test('the first member survey update has a purposeful hero image', function () {
  var surveyUpdate = fs.readFileSync(
    path.join(updatesDirectory, 'first-member-survey.md'),
    'utf8'
  );

  assert.match(surveyUpdate, /heroImage: \/images\/september-survey-2026-hero\.jpg/);
  assert.match(surveyUpdate, /heroImageAlt:\s*["'][^"']+['"]/);
});
