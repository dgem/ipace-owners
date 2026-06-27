const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');

function read(path) {
  return readFileSync(resolve(repoRoot, path), 'utf8');
}

test('runtime declarations use the supported Node and Go production lines', function () {
  const packageJson = JSON.parse(read('package.json'));
  const makefile = read('Makefile');
  const goMod = read('functions/firebase-go/go.mod');

  assert.equal(read('.nvmrc').trim(), '24');
  assert.equal(packageJson.engines.node, '>=24 <25');
  assert.match(packageJson.devDependencies['firebase-tools'], /^\^15\./);
  assert.doesNotMatch(makefile, /go123/);
  assert.match(makefile, /--runtime=go126/);
  assert.match(goMod, /^go 1\.26$/m);
});

test('deployment uses a single Go API Function entrypoint', function () {
  const firebase = read('firebase.json');
  const makefile = read('Makefile');
  const main = read('functions/firebase-go/main.go');

  assert.match(makefile, /FUNCTION_ENTRYPOINTS \?= Api/);
  assert.match(firebase, /"source": "\/api\/\*\*"/);
  assert.match(firebase, /"functionId": "Api"/);
  assert.doesNotMatch(firebase, /"functionId": "SendMagicLink"/);
  assert.doesNotMatch(firebase, /"functionId": "SubmitJoin"/);
  assert.match(main, /functions\.HTTP\("Api", Api\)/);
  assert.match(main, /case "\/api\/send-magic-link":/);
  assert.match(main, /case "\/api\/public-stats":/);
});


test('OpenTofu uses the current provider major and committed exact selections', function () {
  const rootVersions = read('infra/opentofu/env/versions.tf');
  const moduleVersions = read('infra/opentofu/modules/ipace-owners/versions.tf');
  const moduleMain = read('infra/opentofu/modules/ipace-owners/main.tf');
  const lockfile = read('infra/opentofu/env/.terraform.lock.hcl');

  for (const versions of [rootVersions, moduleVersions]) {
    assert.match(versions, /required_version = ">= 1\.12\.3, < 2\.0\.0"/);
    assert.equal((versions.match(/version = "~> 7\.0"/g) || []).length, 2);
  }
  assert.match(lockfile, /provider "registry\.opentofu\.org\/hashicorp\/google"/);
  assert.match(lockfile, /version\s+= "7\.\d+\.\d+"/);
  assert.match(moduleMain, /phone_number \{\s+enabled\s+= false\s+test_phone_numbers = \{\}/);
});

test('Dependabot checks every maintained dependency ecosystem weekly', function () {
  const dependabot = read('.github/dependabot.yml');

  for (const ecosystem of ['npm', 'gomod', 'github-actions', 'terraform']) {
    assert.match(dependabot, new RegExp(`package-ecosystem: ${ecosystem}`));
  }
  assert.equal((dependabot.match(/interval: weekly/g) || []).length, 4);
});
