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
  assert.match(workflow, /rules_file_name: \.zap\/rules\.tsv/);
  assert.match(workflow, /fail_action: true/);
  assert.match(workflow, /allow_issue_writing: false/);

  var rules = read('.zap/rules.tsv');
  assert.match(rules, /^10015\tIGNORE\t/m);
  assert.match(rules, /^90004\tIGNORE\t/m);
});

test('external pull requests validate without receiving staging deployment permissions', function () {
  var workflow = read('.github/workflows/gcp-firebase-staging.yml');
  var validateIndex = workflow.indexOf('  validate:');
  var deployIndex = workflow.indexOf('  deploy-preview:');

  assert.ok(validateIndex >= 0 && deployIndex > validateIndex);
  assert.match(workflow, /permissions:\n[ ]{2}contents: read\n\njobs:/);
  assert.match(workflow, /deploy-preview:\n[ ]{4}needs: validate\n[ ]{4}if: github\.event\.pull_request\.head\.repo\.full_name == github\.repository/);
  assert.match(workflow, /deploy-preview:[\s\S]*concurrency:\n[ ]{6}group: firebase-staging-deploy/);
  assert.match(workflow, /deploy-preview:[\s\S]*permissions:\n[ ]{6}contents: read\n[ ]{6}id-token: write\n[ ]{6}pull-requests: write/);
  assert.match(workflow, /validate:[\s\S]*name: Build site without deployment credentials\n[ ]{8}run: make build/);
});

test('passwordless login fallback never places email addresses in page URLs', function () {
  var gate = read('src/_includes/partials/auth-login-gate.njk');

  assert.match(gate, /<form[^>]+method="POST"[^>]+action="\/api\/send-magic-link"/);
});

test('Dependabot groups compatible updates on a weekly schedule', function () {
  var dependabot = read('.github/dependabot.yml');

  assert.equal((dependabot.match(/interval: weekly/g) || []).length, 4);
  assert.match(dependabot, /npm-minor-and-patch/);
  assert.match(dependabot, /go-minor-and-patch/);
  assert.match(dependabot, /github-actions:[\s\S]*patterns:/);
  assert.match(dependabot, /opentofu-minor-and-patch/);
});
