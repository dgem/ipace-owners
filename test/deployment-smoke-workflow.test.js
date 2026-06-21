const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const workflowPath = resolve(__dirname, '../.github/workflows/deployment-smoke.yml');

test('deployment smoke workflow only runs against owned Firebase/site URLs', function () {
  const workflow = readFileSync(workflowPath, 'utf8');

  assert.match(workflow, /contains\(github\.event\.deployment_status\.target_url, 'ipace-owners\.org'\)/);
  assert.match(workflow, /contains\(github\.event\.deployment_status\.target_url, '\.web\.app'\)/);
  assert.match(workflow, /contains\(github\.event\.deployment_status\.target_url, '\.firebaseapp\.com'\)/);
  assert.doesNotMatch(workflow, /contains\(github\.event\.deployment_status\.target_url, 'github\.com'\)/);
});
