const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const { mkdtempSync, readFileSync, writeFileSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { test } = require('node:test');

const scriptPath = resolve(__dirname, '../scripts/extract-firebase-preview-url.mjs');
const workflowPath = resolve(__dirname, '../.github/workflows/gcp-firebase-staging.yml');
const makefilePath = resolve(__dirname, '../Makefile');

test('extracts the Firebase Hosting preview URL from nested deploy JSON', function () {
  const cwd = mkdtempSync(join(tmpdir(), 'ipace-preview-url-'));
  const jsonPath = join(cwd, 'firebase-preview.json');

  writeFileSync(jsonPath, JSON.stringify({
    status: 'success',
    result: {
      metadata: {
        console: 'https://console.firebase.google.com/project/ipace-owners-staging/hosting',
      },
      sites: {
        default: {
          url: 'https://ipace-owners-staging--pr-15-a1b2c3d4.web.app',
        },
      },
    },
  }));

  const output = execFileSync(process.execPath, [scriptPath, jsonPath], {
    cwd,
    encoding: 'utf8',
    env: {
      ...process.env,
      GITHUB_OUTPUT: '',
    },
  }).trim();

  assert.equal(output, 'https://ipace-owners-staging--pr-15-a1b2c3d4.web.app');
});

test('writes the Firebase Hosting preview URL to GitHub output when available', function () {
  const cwd = mkdtempSync(join(tmpdir(), 'ipace-preview-url-'));
  const jsonPath = join(cwd, 'firebase-preview.json');
  const outputPath = join(cwd, 'github-output.txt');

  writeFileSync(jsonPath, JSON.stringify({
    hosting: {
      url: 'https://ipace-owners-staging--pr-15-a1b2c3d4.web.app',
    },
  }));

  execFileSync(process.execPath, [scriptPath, jsonPath], {
    cwd,
    encoding: 'utf8',
    env: {
      ...process.env,
      GITHUB_OUTPUT: outputPath,
    },
  });

  assert.equal(
    readFileSync(outputPath, 'utf8').trim(),
    'url=https://ipace-owners-staging--pr-15-a1b2c3d4.web.app',
  );
});

test('staging Functions use the current PR preview URL', function () {
  const workflow = readFileSync(workflowPath, 'utf8');
  const firstHostingDeploy = workflow.indexOf('- name: Deploy Firebase Hosting preview');
  const authorizePreview = workflow.indexOf(
    '- name: Authorize Firebase Hosting preview for passwordless sign-in',
  );
  const functionDeploy = workflow.indexOf('- name: Deploy Go Cloud Functions');
  const refreshedHostingDeploy = workflow.indexOf(
    '- name: Refresh Firebase Hosting preview with current Function revisions',
  );

  assert.ok(firstHostingDeploy > -1 && firstHostingDeploy < functionDeploy);
  assert.ok(authorizePreview > firstHostingDeploy && authorizePreview < functionDeploy);
  assert.ok(refreshedHostingDeploy > functionDeploy);
  assert.match(workflow, /run: make authorize-preview-domain/);
  assert.match(workflow, /group: firebase-staging-deploy/);
  assert.match(workflow, /ALLOWED_ORIGINS: \$\{\{ steps\.hosting\.outputs\.url \}\}/);
  assert.match(
    workflow,
    /FIREBASE_EMAIL_CONTINUE_URL: \$\{\{ format\('\{0\}\/account\/', steps\.hosting\.outputs\.url\) \}\}/,
  );
  assert.doesNotMatch(workflow, /FIREBASE_EMAIL_(?:CONTINUE_URL|LINK_DOMAIN)_STAGING/);
});

test('preview deployment reports Firebase CLI diagnostics when deployment fails', function () {
  const makefile = readFileSync(makefilePath, 'utf8');
  const workflow = readFileSync(workflowPath, 'utf8');

  assert.match(makefile, /hosting:channel:deploy[^\n]+--debug/);
  assert.match(makefile, /firebase-access-token-preload\.cjs/);
  assert.match(makefile, /for attempt in 1 2 3/);
  assert.match(makefile, /else \\\n\s+status=\$\$\?;/);
  assert.match(makefile, /sleep \$\$\(\(attempt \* 5\)\)/);
  assert.match(makefile, /cat "\$\$error_log" >&2/);
  assert.match(makefile, /cat "\$\(FIREBASE_PREVIEW_JSON\)" >&2/);
  assert.match(makefile, /tail -80 "\$\$error_log"/);
  assert.match(makefile, /GITHUB_STEP_SUMMARY/);
  assert.match(makefile, /FIREBASE_PREVIEW_ERROR \?= firebase-preview-error\.log/);
  assert.match(workflow, /Firebase CLI diagnostics/);
  assert.match(workflow, /Firebase debug log/);
  assert.match(workflow, /deployFailed && fs\.existsSync\('firebase-debug\.log'\)/);
  assert.match(makefile, /exit "\$\$status"/);
});
