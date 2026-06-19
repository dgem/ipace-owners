const assert = require('node:assert/strict');
const { execFileSync } = require('node:child_process');
const { mkdtempSync, readFileSync, writeFileSync } = require('node:fs');
const { tmpdir } = require('node:os');
const { join, resolve } = require('node:path');
const { test } = require('node:test');

const scriptPath = resolve(__dirname, '../scripts/extract-firebase-preview-url.mjs');

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
