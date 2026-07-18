'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');
var read = function (file) { return fs.readFileSync(path.join(root, file), 'utf8'); };

test('base layout renders shared canonical, social and structured SEO metadata', function () {
  var base = read('src/_includes/layouts/base.njk');
  var seo = read('src/_includes/partials/seo.njk');
  var site = JSON.parse(read('src/_data/site.json'));

  assert.match(base, /include "partials\/seo\.njk"/);
  assert.match(seo, /page\.url == "\/"/);
  assert.match(seo, /description or summary or site\.description/);
  assert.match(seo, /rel="canonical"/);
  assert.match(seo, /name="robots"/);
  assert.match(seo, /property="og:title"/);
  assert.match(seo, /property="og:image:alt"/);
  assert.match(seo, /name="twitter:card" content="summary_large_image"/);
  assert.match(seo, /type="application\/ld\+json"/);
  assert.match(seo, /"@type": "Organization"/);
  assert.match(seo, /"@type": "WebSite"/);
  assert.doesNotMatch(seo, /name="keywords"/);
  assert.equal(site.url, 'https://ipace-owners.org');
  assert.equal(site.socialImage, '/images/ipace-hero.png');
  assert.equal(site.logo, '/images/ipace-owners-logo.png');
});

test('private account, vehicle, member and admin pages opt out of indexing', function () {
  [
    'src/member/account.njk',
    'src/member/submit-vehicle-data.njk',
    'src/member/dashboard.njk',
    'src/admin/review-queue.njk',
  ].forEach(function (file) {
    assert.match(read(file), /noindex: true/, file + ' should be noindex');
  });
});

test('site launch update exposes article metadata', function () {
  var update = read('src/updates/site-launch.md');
  var seo = read('src/_includes/partials/seo.njk');

  assert.match(update, /seoType: article/);
  assert.match(seo, /article:published_time/);
});
