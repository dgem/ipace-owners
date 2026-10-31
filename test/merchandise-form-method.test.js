const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('shop delivery details use POST so scanners and no-script forms do not place PII in URLs', function () {
  const shop = fs.readFileSync(path.join(__dirname, '../src/shop.njk'), 'utf8');
  const form = shop.match(/<form\b[^>]*data-shop-form[^>]*>/);

  assert.ok(form, 'the shop delivery form exists');
  assert.match(form[0], /\bmethod="post"/i);
  assert.doesNotMatch(form[0], /\bmethod="get"/i);
});
