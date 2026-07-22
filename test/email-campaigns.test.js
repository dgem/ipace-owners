const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const page = fs.readFileSync(path.join(root, 'src/admin/email-campaigns.njk'), 'utf8');
const script = fs.readFileSync(path.join(root, 'src/assets/js/email-campaigns.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');

test('admin email campaign page is gated and describes bounded sending', function () {
  assert.match(page, /data-admin-container/);
  assert.match(page, /data-admin-content hidden/);
  assert.match(page, /without revealing addresses/i);
  assert.match(page, /batches of 10/i);
  assert.match(layout, /emailCampaigns[\s\S]*email-campaigns\.js/);
  assert.ok(page.indexOf('Admin navigation') < page.indexOf('Deliberately hard to send'));
});

test('admin workspaces put navigation before their main content', function () {
  const outreach = fs.readFileSync(path.join(root, 'src/admin/outreach.njk'), 'utf8');
  const review = fs.readFileSync(path.join(root, 'src/admin/review-queue.njk'), 'utf8');
  assert.ok(outreach.indexOf('Admin navigation') < outreach.indexOf('Human controlled by design'));
  assert.ok(review.indexOf('Admin navigation') < review.indexOf('Admin review queue — live data'));
});

test('email campaign browser sends tokens and explicit confirmation data', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(script, /\/api\/admin\/reengagement-preview/);
  assert.match(script, /\/api\/admin\/reengagement-send/);
  assert.match(script, /expectedEligible: current\.eligible/);
  assert.match(script, /confirmation: confirmInput\.value/);
  assert.doesNotMatch(script, /recipient|emailAddress|\.email\b/i);
});

test('portable homepage copy uses production links and live-value placeholders', function () {
  const copy = fs.readFileSync(path.join(root, 'docs/homepage-copy.md'), 'utf8');
  const joinUrl = ['https:', '', 'ipace-owners.org', 'join', ''].join('/');
  assert.ok(copy.includes(joinUrl));
  assert.match(copy, /\[Current owners joined\]/);
  assert.match(copy, /I-PACE owners working together for fair outcomes/);
  assert.doesNotMatch(copy, /\{[%{]/);
});
