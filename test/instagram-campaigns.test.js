const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const page = fs.readFileSync(path.join(root, 'src/admin/instagram-campaigns.njk'), 'utf8');
const script = fs.readFileSync(path.join(root, 'src/assets/js/instagram-campaigns.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');

test('Instagram campaign follows the gated preview and exact-confirmation pattern', function () {
  assert.match(page, /data-admin-container/);
  assert.match(page, /Previewing never posts/i);
  assert.match(page, /normal, controlled stop/i);
  assert.match(page, /data-instagram-publish-button disabled/);
  assert.match(page, /\/api\/admin\/instagram-preview/);
  assert.match(page, /\/api\/admin\/instagram-publish/);
  assert.match(layout, /instagramCampaigns[\s\S]*instagram-campaigns\.js/);
});

test('browser sends only an admin token and the reviewed draft', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(script, /mediaReviewed: mediaReviewed\.checked/);
  assert.match(script, /confirmation: confirm\.value/);
  assert.match(script, /invalidate\(\)/);
  assert.doesNotMatch(script, /INSTAGRAM_ACCESS_TOKEN/);
});
