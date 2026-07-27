const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const page = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');
const script = fs.readFileSync(path.join(root, 'src/assets/js/admin-campaign-summary.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');

test('Admin home exposes a claim-gated cross-channel campaign summary', function () {
  assert.match(page, /data-admin-container/);
  assert.match(page, /data-campaign-summary/);
  assert.match(page, /\/api\/admin\/campaign-summary/);
  assert.match(page, /Instagram engagement comes from Meta/);
  assert.match(layout, /adminDashboard[\s\S]*admin-campaign-summary\.js/);
});

test('campaign summary uses the Firebase token and renders provider values as text', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(script, /Authorization.*Bearer/);
  assert.match(script, /replaceChildren/);
  assert.match(script, /textContent/);
  assert.doesNotMatch(script, /innerHTML/);
});
