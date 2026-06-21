const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');
const scriptPath = resolve(repoRoot, 'scripts/infra-env.sh');

function runConfig(environment, tfvars) {
  return spawnSync(scriptPath, ['config'], {
    cwd: repoRoot,
    encoding: 'utf8',
    env: {
      ...process.env,
      ENV: environment,
      TFVARS: tfvars,
    },
  });
}

test('resolves staging infrastructure configuration without cloud access', function () {
  const result = runConfig('staging', 'staging.tfvars.example');

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Environment: staging/);
  assert.match(result.stdout, /Project: ipace-owners-staging/);
  assert.match(result.stdout, /Quota project: ipace-owners-staging/);
  assert.match(result.stdout, /Workspace: staging/);
  assert.match(result.stdout, /staging\.tfvars\.example/);
});

test('resolves production infrastructure configuration without cloud access', function () {
  const result = runConfig('production', 'production.tfvars.example');

  assert.equal(result.status, 0, result.stderr);
  assert.match(result.stdout, /Environment: production/);
  assert.match(result.stdout, /Project: ipace-owners-production/);
  assert.match(result.stdout, /Workspace: production/);
});

test('requires an explicit infrastructure environment', function () {
  const result = runConfig('', 'staging.tfvars.example');

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /ENV must be staging or production/);
});

test('rejects tfvars for a different environment', function () {
  const result = runConfig('production', 'staging.tfvars.example');

  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /does not match environment/);
});
