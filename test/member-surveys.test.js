'use strict';

const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.join(__dirname, '..');

test('admin dashboard exposes the implemented member surveys workspace', function () {
  const dashboard = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');
  const surveys = fs.readFileSync(path.join(root, 'src/admin/surveys.njk'), 'utf8');

  assert.match(dashboard, /href="\/admin\/surveys\/">Member Surveys<\/a>/);
  assert.match(surveys, /permalink: \/admin\/surveys\//);
  assert.match(surveys, /data-admin-content hidden data-survey-admin/);
});

test('surveys support public Markdown copy, multiple text options, and seven-day defaults', function () {
  const admin = fs.readFileSync(path.join(root, 'src/admin/surveys.njk'), 'utf8');
  const script = fs.readFileSync(path.join(root, 'src/assets/js/surveys.js'), 'utf8');

  assert.match(admin, /data-survey-description/);
  assert.match(admin, /data-survey-cta/);
  assert.match(script, /maxlength="2000"/);
  assert.match(script, /function markdown\(v\)/);
  assert.match(script, /end\.setDate\(end\.getDate\(\) \+ 6\)/);
  assert.match(script, /textByOption/);
});
