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
  assert.match(page, /Send registration reminder emails/);
  assert.match(page, /What each recipient will receive/);
  assert.doesNotMatch(page, /data-campaign-send hidden/);
  assert.match(page, /data-campaign-send-button disabled/);
  assert.match(layout, /emailCampaigns[\s\S]*email-campaigns\.js/);
  assert.doesNotMatch(page, /Admin navigation/);
});

test('admin navigation lives in the claim-gated site header, not page content', function () {
  const outreach = fs.readFileSync(path.join(root, 'src/admin/outreach.njk'), 'utf8');
  const review = fs.readFileSync(path.join(root, 'src/admin/review-queue.njk'), 'utf8');
  const header = fs.readFileSync(path.join(root, 'src/_includes/partials/header.njk'), 'utf8');
  assert.doesNotMatch(outreach, /Admin navigation/);
  assert.doesNotMatch(review, /Admin navigation/);
  assert.match(header, /site-admin-nav[\s\S]*navigation\.admin/);
});

test('admin index is a gated dashboard of implemented tools', function () {
  const dashboard = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');
  assert.match(dashboard, /data-admin-container/);
  assert.match(dashboard, /data-admin-content hidden/);
  assert.match(dashboard, /Open review queue/);
  assert.match(dashboard, /Open outreach assistant/);
  assert.match(dashboard, /Open email campaigns/);
  assert.match(dashboard, /not linked prematurely/);
});

test('email campaign browser sends tokens and explicit confirmation data', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(page, /\/api\/admin\/reengagement-preview/);
  assert.match(page, /\/api\/admin\/reengagement-send/);
  assert.match(page, /\/api\/admin\/member-referral-preview/);
  assert.match(page, /\/api\/admin\/member-referral-send/);
  assert.match(script, /expectedEligible: current\.eligible/);
  assert.match(script, /confirmation: confirmInput\.value/);
  assert.match(script, /emailText\.textContent = data\.emailPreview\.text/);
  assert.match(script, /emailPreview\.hidden = false/);
  assert.doesNotMatch(script, /recipient|emailAddress|\.email\b/i);
});

test('member referral campaign is clear and previews monochrome sharing actions', function () {
  const css = fs.readFileSync(path.join(root, 'src/assets/css/site.css'), 'utf8');
  assert.match(page, /Ask members to invite one more I-PACE owner/);
  assert.match(page, /registered members whose Join record includes contact consent/);
  assert.match(script, /data\.emailPreview\.shares/);
  assert.match(css, /\.email-preview__share-mark[\s\S]*grayscale\(1\)/);
});

test('portable homepage copy uses production links and live-value placeholders', function () {
  const copy = fs.readFileSync(path.join(root, 'docs/homepage-copy.md'), 'utf8');
  const joinUrl = ['https:', '', 'ipace-owners.org', 'join', ''].join('/');
  assert.ok(copy.includes(joinUrl));
  assert.match(copy, /\[Current owners joined\]/);
  assert.match(copy, /I-PACE owners working together for fair outcomes/);
  assert.doesNotMatch(copy, /\{[%{]/);
});
