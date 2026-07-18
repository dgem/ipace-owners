'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

function read(relativePath) {
  return fs.readFileSync(path.join(repoRoot, relativePath), 'utf8');
}

test('security workflow runs SAST and dependency vulnerability checks', function () {
  var workflow = read('.github/workflows/security.yml');

  assert.match(workflow, /cron: "23 5 \* \* 1"/);
  assert.match(workflow, /actions\/dependency-review-action@v5/);
  assert.match(workflow, /fail-on-severity: moderate/);
  assert.match(workflow, /run: make audit-node/);
  assert.match(workflow, /run: make audit-go/);
  assert.match(workflow, /language: actions[\s\S]*language: javascript-typescript[\s\S]*language: go/);
  assert.match(workflow, /github\/codeql-action\/init@v4/);
  assert.match(workflow, /queries: security-extended/);
});

test('Makefile exposes reproducible Node and Go vulnerability audits', function () {
  var makefile = read('Makefile');

  assert.match(makefile, /GOVULNCHECK_VERSION \?= v1\.6\.0/);
  assert.match(makefile, /audit: audit-node audit-go/);
  assert.match(makefile, /npm audit --audit-level=high/);
  assert.match(makefile, /govulncheck@\$\(GOVULNCHECK_VERSION\)/);
});

test('preview deployment runs a blocking passive OWASP ZAP scan', function () {
  var workflow = read('.github/workflows/gcp-firebase-staging.yml');
  var smokeIndex = workflow.indexOf('name: Smoke test preview');
  var zapIndex = workflow.indexOf('name: OWASP ZAP baseline scan');

  assert.ok(smokeIndex >= 0 && zapIndex > smokeIndex);
  assert.match(workflow, /zaproxy\/action-baseline@v0\.15\.0/);
  assert.match(workflow, /docker_name: ghcr\.io\/zaproxy\/zaproxy:2\.17\.0/);
  assert.match(workflow, /fail_action: true/);
  assert.match(workflow, /allow_issue_writing: false/);
});

test('Dependabot groups compatible updates on a weekly schedule', function () {
  var dependabot = read('.github/dependabot.yml');

  assert.equal((dependabot.match(/interval: weekly/g) || []).length, 4);
  assert.match(dependabot, /npm-minor-and-patch/);
  assert.match(dependabot, /go-minor-and-patch/);
  assert.match(dependabot, /github-actions:[\s\S]*patterns:/);
  assert.match(dependabot, /opentofu-minor-and-patch/);
});
