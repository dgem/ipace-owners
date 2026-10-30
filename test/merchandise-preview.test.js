const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('shop keeps a local preview basket available when the catalogue API fails', function () {
  const shop = fs.readFileSync(path.join(__dirname, '../src/shop.njk'), 'utf8');
  const script = fs.readFileSync(path.join(__dirname, '../src/assets/js/merchandise.js'), 'utf8');

  assert.match(shop, /data-shop-checkout-note/);
  assert.match(shop, /data-shop-form[^>]*hidden/);
  assert.match(script, /function usePreviewProducts\(\)/);
  assert.match(script, /You can still add previews to your basket/);
  assert.match(script, /form\.hidden = !ready/);
  assert.match(script, /if \(busy \|\| !ready \|\| !basket\.length\) return/);
  assert.doesNotMatch(script, /catch \(function \(error\) \{ status\.textContent = error\.message; \}\);/);
});
