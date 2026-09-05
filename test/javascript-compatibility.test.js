const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const { test } = require('node:test');

const repoRoot = resolve(__dirname, '..');

test('browser JavaScript linting rejects trailing function commas without constraining build tooling', function () {
  const config = readFileSync(resolve(repoRoot, 'eslint.config.mjs'), 'utf8');

  assert.match(config, /files: \['src\/assets\/js\/\*\*\/\*\.js'\]/);
  assert.match(config, /functions: 'never'/);
  assert.match(config, /arrays: 'ignore'/);
  assert.match(config, /objects: 'ignore'/);
  assert.doesNotMatch(config, /ecmaVersion: 2015/);
});
