const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('merchandise shows a clear unavailable state instead of inviting shoppers to build a fake basket', function () {
  const shop = fs.readFileSync(path.join(__dirname, '../src/shop.njk'), 'utf8');
  const script = fs.readFileSync(path.join(__dirname, '../src/assets/js/merchandise.js'), 'utf8');
  const navigation = JSON.parse(fs.readFileSync(path.join(__dirname, '../src/_data/navigation.json'), 'utf8'));

  assert.match(shop, /data-shop-checkout-note/);
  assert.match(shop, /orders and payments are not available in this environment yet/i);
  assert.match(shop, /data-shop-form[^>]*hidden/);
  assert.ok(shop.indexOf('data-shop-basket') < shop.indexOf('id="merchandise-catalogue"'));
  assert.equal(navigation.public.find(function (item) { return item.url === '/shop/'; }).label, 'Merchandise');
  assert.match(script, /function usePreviewProducts\(\)/);
  assert.match(script, /disabled>Ordering unavailable/);
  assert.match(script, /checkout is not enabled here yet/i);
  assert.match(script, /form\.hidden = !ready/);
  assert.match(script, /if \(busy \|\| !ready \|\| !basket\.length\) return/);
  assert.match(script, /if \(wasEmpty\) \{ root\.querySelector\('#basket-title'\)\.focus\(\); \}/);
  assert.doesNotMatch(script, /catch \(function \(error\) \{ status\.textContent = error\.message; \}\);/);
});
