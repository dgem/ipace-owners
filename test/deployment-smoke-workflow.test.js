const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const { existsSync, readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const deploymentSmokeWorkflowPath = resolve(__dirname, '../.github/workflows/deployment-smoke.yml');
const stagingWorkflowPath = resolve(__dirname, '../.github/workflows/gcp-firebase-staging.yml');
const productionWorkflowPath = resolve(__dirname, '../.github/workflows/gcp-firebase-production.yml');
const firebaseAccessTokenPreloadPath = resolve(
  __dirname,
  '../scripts/firebase-access-token-preload.cjs',
);

test('deployment smoke tests run inside Firebase deploy workflows', function () {
  const stagingWorkflow = readFileSync(stagingWorkflowPath, 'utf8');
  const productionWorkflow = readFileSync(productionWorkflowPath, 'utf8');

  assert.equal(existsSync(deploymentSmokeWorkflowPath), false);
  assert.match(stagingWorkflow, /name: Smoke test preview/);
  assert.match(stagingWorkflow, /SMOKE_BASE_URL: \$\{\{ steps\.hosting\.outputs\.url \}\}/);
  assert.match(stagingWorkflow, /run: make smoke/);
  assert.match(productionWorkflow, /name: Smoke test production/);
  assert.match(productionWorkflow, /SMOKE_BASE_URL: https:\/\/ipace-owners\.org/);
  assert.match(productionWorkflow, /run: make smoke/);
});

test('Firebase deploy workflows skip Function deploys when backend is unchanged', function () {
  const stagingWorkflow = readFileSync(stagingWorkflowPath, 'utf8');
  const productionWorkflow = readFileSync(productionWorkflowPath, 'utf8');

  for (const workflow of [stagingWorkflow, productionWorkflow]) {
    assert.match(workflow, /name: Detect backend deploy changes/);
    assert.match(workflow, /functions\/firebase-go\//);
    assert.match(workflow, /firebase\\.json/);
    assert.match(workflow, /if: steps\.backend\.outputs\.deploy == 'true'/);
  }
  assert.match(stagingWorkflow, /name: Refresh Firebase Hosting preview with current Function revisions/);
  assert.match(stagingWorkflow, /if: steps\.backend\.outputs\.deploy == 'true'/);
});

test('Firebase deploy workflows verify keyless GCP credentials before deployment', function () {
  const stagingWorkflow = readFileSync(stagingWorkflowPath, 'utf8');
  const productionWorkflow = readFileSync(productionWorkflowPath, 'utf8');

  for (const workflow of [stagingWorkflow, productionWorkflow]) {
    assert.match(workflow, /token_format: access_token/);
    assert.match(workflow, /create_credentials_file: true/);
    assert.match(workflow, /export_environment_variables: true/);
    assert.match(workflow, /run: gcloud auth print-access-token >\/dev\/null/);
    assert.match(workflow, /FIREBASE_CLI_ACCESS_TOKEN: \$\{\{ steps\.gcp-auth\.outputs\.access_token \}\}/);
  }
});

test('Firebase CLI preload reuses the short-lived GitHub access token', function () {
  const command = [
    "const { createRequire } = require('node:module');",
    "const firebaseRequire = createRequire(require.resolve('firebase-tools/package.json'));",
    "const { GoogleAuth } = firebaseRequire('google-auth-library');",
    "new GoogleAuth().getAccessToken().then((token) => process.stdout.write(token));",
  ].join(' ');

  const output = execFileSync(process.execPath, ['-e', command], {
    cwd: resolve(__dirname, '..'),
    encoding: 'utf8',
    env: {
      ...process.env,
      FIREBASE_CLI_ACCESS_TOKEN: 'short-lived-test-token',
      NODE_OPTIONS: `--require=${firebaseAccessTokenPreloadPath}`,
    },
  });

  assert.equal(output, 'short-lived-test-token');
});
