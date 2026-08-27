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
