const assert = require('node:assert/strict');
const { spawnSync } = require('node:child_process');
const { resolve } = require('node:path');
const { test } = require('node:test');
const { readFileSync } = require('node:fs');

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

test('local infra commands source only ignored environment-scoped application and commerce credentials', function () {
  const script = readFileSync(scriptPath, 'utf8');
  const example = readFileSync(resolve(repoRoot, '.env.example'), 'utf8');
  const ignore = readFileSync(resolve(repoRoot, '.gitignore'), 'utf8');

  assert.match(ignore, /^\.env$/m);
  assert.match(script, /source "\$\{repo_root\}\/\.env"/);
  assert.match(script, /scoped_name=.*secret_name.*secret_suffix/);
  assert.match(script, /Never fall back to an unscoped secret from another shell\/session/);
  for (const name of ['VIN_PEPPER', 'RESEND_API_KEY', 'IDEAL_POSTCODES_API_KEY', 'INSTAGRAM_ACCESS_TOKEN', 'PRINTIFY_API_TOKEN', 'PRINTIFY_WEBHOOK_SECRET', 'STRIPE_SECRET_KEY', 'STRIPE_WEBHOOK_SECRET']) {
    assert.match(script, new RegExp(`for secret_name in[^\\n]*${name}`));
    assert.match(example, new RegExp(`${name}_STAGING`));
    assert.match(example, new RegExp(`${name}_PRODUCTION`));
  }
  assert.match(example, /STRIPE_SECRET_KEY_STAGING=sk_test_/);
  assert.match(example, /STRIPE_SECRET_KEY_PRODUCTION=sk_live_/);
  assert.match(example, /PRINTIFY_WEBHOOK_SECRET_STAGING/);
  assert.match(example, /PRINTIFY_WEBHOOK_SECRET_PRODUCTION/);
});

test('OpenTofu owns merchandise deployment variables without taking provider secrets as inputs', function () {
  const actions = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/github-actions.tf'), 'utf8');
  const envVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/env/variables.tf'), 'utf8');
  const moduleVariables = readFileSync(resolve(repoRoot, 'infra/opentofu/modules/ipace-owners/variables.tf'), 'utf8');
  const ignore = readFileSync(resolve(repoRoot, '.gitignore'), 'utf8');

  assert.match(actions, /"MERCH_CONFIG_\$\{local\.github_actions_suffix\}"\s*=\s*var\.merchandise_config/);
  assert.match(actions, /"MERCH_SECRETS_ENABLED_\$\{local\.github_actions_suffix\}"\s*=\s*tostring\(var\.merchandise_secrets_enabled\)/);
  assert.match(envVariables, /variable "merchandise_config"/);
  assert.match(envVariables, /variable "merchandise_secrets_enabled"/);
  assert.match(moduleVariables, /Never put provider credentials or webhook signing secrets in this value/);
  assert.match(ignore, /merchandise-config\.staging\.json/);
  assert.match(ignore, /merchandise-config\.production\.json/);
  assert.doesNotMatch(envVariables, /variable "(?:stripe_secret|printify_api_token)/i);
});
