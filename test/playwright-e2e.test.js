const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const { resolve } = require('node:path');
const test = require('node:test');

function read(path) {
  return readFileSync(resolve(__dirname, '..', path), 'utf8');
}

test('Playwright runs responsive coverage across desktop and mobile browsers', function () {
  const config = read('playwright.config.ts');
  const workflow = read('.github/workflows/playwright.yml');

  for (const project of [
    'responsive-chromium',
    'responsive-firefox',
    'responsive-webkit',
    'responsive-mobile-chrome',
    'responsive-mobile-safari',
  ]) {
    assert.match(config, new RegExp(`name: '${project}'`));
  }

  assert.match(config, /webServer:/);
  assert.match(workflow, /make test-e2e-responsive/);
  assert.match(workflow, /install --with-deps chromium firefox webkit/);
});

test('the live magic-link journey is opt-in and does not retain sensitive diagnostics', function () {
  const config = read('playwright.config.ts');
  const spec = read('tests/member-magic-link.spec.ts');
  const workflow = read('.github/workflows/gcp-firebase-staging.yml');

  assert.match(config, /name: 'member-magic-link'/);
  assert.match(config, /trace: 'off'/);
  assert.match(config, /screenshot: 'off'/);
  assert.match(spec, /E2E_RESEND_API_KEY/);
  assert.match(spec, /E2E_RESEND_INBOX/);
  assert.match(spec, /'\/emails\/receiving'/);
  assert.match(spec, /test\.skip\(!hasLiveAuthConfiguration/);
  assert.doesNotMatch(spec, /console\.(?:log|error|warn)/);
  assert.match(workflow, /PLAYWRIGHT_AUTH_E2E_ENABLED == 'true'/);
  assert.match(workflow, /E2E_RESEND_INBOX_STAGING/);
  assert.match(workflow, /E2E_RESEND_API_KEY_STAGING/);
  assert.match(workflow, /run: make test-e2e-auth/);
});
