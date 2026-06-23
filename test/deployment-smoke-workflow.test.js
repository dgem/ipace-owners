const assert = require('node:assert/strict');
const { existsSync, readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const deploymentSmokeWorkflowPath = resolve(__dirname, '../.github/workflows/deployment-smoke.yml');
const stagingWorkflowPath = resolve(__dirname, '../.github/workflows/gcp-firebase-staging.yml');
const productionWorkflowPath = resolve(__dirname, '../.github/workflows/gcp-firebase-production.yml');

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
