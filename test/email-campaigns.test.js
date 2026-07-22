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
