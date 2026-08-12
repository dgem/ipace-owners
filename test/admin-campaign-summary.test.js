const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const page = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');
const script = fs.readFileSync(path.join(root, 'src/assets/js/admin-campaign-summary.js'), 'utf8');
const statsScript = fs.readFileSync(path.join(root, 'src/assets/js/admin-stats.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');

test('Admin home exposes a claim-gated cross-channel campaign summary', function () {
  assert.match(page, /data-admin-container/);
  assert.match(page, /data-campaign-summary/);
  assert.match(page, /admin-stats-section/);
  assert.match(page, /admin-timeline/);
  assert.match(page, /\/api\/admin\/campaign-summary/);
  assert.match(page, /Email delivery feedback is reconciled with Resend/);
  assert.match(page, /Instagram engagement comes from Meta/);
  assert.match(layout, /adminDashboard[\s\S]*admin-campaign-summary\.js/);
  assert.match(layout, /adminDashboard[\s\S]*admin-stats\.js/);
});

test('campaign summary uses the Firebase token and renders provider values as text', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(script, /Authorization.*Bearer/);
  assert.match(script, /replaceChildren/);
  assert.match(script, /textContent/);
  assert.match(script, /undeliverable/);
  assert.match(script, /bounced/);
  assert.match(script, /delayed/);
  assert.doesNotMatch(script, /innerHTML/);
});

test('admin statistics load only after admin visibility with an identity token', function () {
  assert.match(statsScript, /ipaceGetIdentityToken/);
  assert.match(statsScript, /Authorization.*Bearer/);
  assert.match(statsScript, /attributeFilter: \['hidden'\]/);
  assert.match(statsScript, /renderStatCards\(container, '\[data-member-stats\]'/);
  assert.match(statsScript, /renderStatCards\(container, '\[data-vehicle-stats\]'/);
  assert.doesNotMatch(statsScript, /container\.innerHTML\s*=/);
});
