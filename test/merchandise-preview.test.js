const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('merchandise storefront and navigation use the server-resolved audience gate', function () {
  const shop = fs.readFileSync(path.join(__dirname, '../src/shop.njk'), 'utf8');
  const script = fs.readFileSync(path.join(__dirname, '../src/assets/js/merchandise.js'), 'utf8');
  const navigation = JSON.parse(fs.readFileSync(path.join(__dirname, '../src/_data/navigation.json'), 'utf8'));

  assert.match(shop, /data-shop-checkout-note/);
  assert.match(shop, /data-member-feature="merchandise" hidden/);
  assert.match(shop, /data-member-feature-denied="merchandise"/);
  assert.match(shop, /data-shop-form[^>]*hidden/);
  assert.ok(shop.indexOf('data-shop-basket') < shop.indexOf('id="merchandise-catalogue"'));
  const merchandiseNav = navigation.public.find(function (item) { return item.url === '/shop/'; });
  assert.equal(merchandiseNav.label, 'Merchandise');
  assert.equal(merchandiseNav.feature, 'merchandise');
  assert.match(script, /disabled>Ordering unavailable/);
  assert.match(script, /checkout is not enabled here yet/i);
  assert.match(script, /form\.hidden = !ready/);
  assert.match(script, /if \(busy \|\| !ready \|\| !basket\.length\) return/);
  assert.match(script, /if \(wasEmpty\) \{ root\.querySelector\('#basket-title'\)\.focus\(\); \}/);
  assert.match(script, /X-Firebase-ID-Token/);
  assert.match(script, /Authorization: 'Bearer ' \+ receipt/);
  assert.match(script, /Merchandise is unavailable right now/);
  assert.doesNotMatch(shop, /for product in merchandisePreview/);
});
