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
  assert.ok(surveys.indexOf('Existing surveys') < surveys.indexOf('Create or edit a survey'));
  assert.match(surveys, /survey-editor__title[\s\S]*type="text"/);
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
  assert.match(script, /document\.addEventListener\('admin:data', start\)/);
  assert.match(script, /window\.setInterval\(load, 30000\)/);
});

test('member dashboard prominently links members to current and closed surveys', function () {
  const dashboard = fs.readFileSync(path.join(root, 'src/member/dashboard.njk'), 'utf8');
  const account = fs.readFileSync(path.join(root, 'src/member/account.njk'), 'utf8');
  const script = fs.readFileSync(path.join(root, 'src/assets/js/surveys.js'), 'utf8');

  assert.match(dashboard, /data-member-survey-summary/);
  assert.match(dashboard, /surveys: true/);
  assert.match(script, /Help steer our discussions with JLR/);
  assert.match(script, /filter=closed/);
  assert.match(script, /href="\/member\/surveys\/"/);
  assert.match(account, /data-member-survey-summary/);
  assert.match(account, /surveys: true/);
});

test('member survey choices are structured as prominent selectable cards', function () {
  const script = fs.readFileSync(path.join(root, 'src/assets/js/surveys.js'), 'utf8');
  const css = fs.readFileSync(path.join(root, 'src/assets/css/site.css'), 'utf8');

  assert.match(script, /survey-choice__number/);
  assert.match(script, /Select every outcome you would support/);
  assert.match(script, /survey-results__header/);
  assert.match(css, /\.survey-choice:has\(input:checked\)/);
  assert.match(css, /\.survey-member-card__title/);
});

test('survey directory filters its date-ordered list and results follow the blind-vote rule', function () {
  const script = fs.readFileSync(path.join(root, 'src/assets/js/surveys.js'), 'utf8');
  const backend = fs.readFileSync(path.join(root, 'functions/firebase-go/surveys.go'), 'utf8');
  const results = fs.readFileSync(path.join(root, 'src/member/survey-results.njk'), 'utf8');
  const response = fs.readFileSync(path.join(root, 'src/member/survey-response.njk'), 'utf8');

  assert.match(backend, /memberMayViewSurveyResults/);
  assert.match(backend, /surveyIsPublished/);
  assert.match(script, /filter === 'closed'/);
  assert.match(script, /survey-response\/\?id=/);
  assert.match(script, /survey-results\/\?id=/);
  assert.match(script, /window\.location\.assign/);
  assert.match(script, /!result\.canRespond/);
  assert.match(results, /data-member-survey-results/);
  assert.match(response, /data-member-survey-response/);
});

test('survey analysis is an admin-only page with masked-response CSV download', function () {
  const admin = fs.readFileSync(path.join(root, 'src/admin/surveys.njk'), 'utf8');
  const analysis = fs.readFileSync(path.join(root, 'src/admin/survey-results.njk'), 'utf8');
  const script = fs.readFileSync(path.join(root, 'src/assets/js/surveys.js'), 'utf8');
  const backend = fs.readFileSync(path.join(root, 'functions/firebase-go/surveys.go'), 'utf8');

  assert.match(admin, /admin-only results/);
  assert.match(script, /\/admin\/survey-results\/\?id=/);
  assert.match(analysis, /data-admin-survey-results/);
  assert.match(analysis, /data-admin-container/);
  assert.match(script, /format=csv/);
  assert.match(backend, /func AdminSurveyResults/);
  assert.match(backend, /maskedEmail/);
});
