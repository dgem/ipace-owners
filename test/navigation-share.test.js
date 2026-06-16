'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('header swaps Join navigation for Account when signed in', function () {
  var header = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/header.njk'), 'utf8');
  var mobileNav = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/mobile-nav.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');

  assert.match(header, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.match(header, /href="\/account\/"[\s\S]*data-requires-auth/);
  assert.match(mobileNav, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.match(mobileNav, /href="\/account\/"[\s\S]*data-requires-auth/);
  assert.match(identityJs, /setVisibility\('\[data-requires-auth\]', true\)/);
  assert.match(identityJs, /setVisibility\('\[data-requires-guest\]', false\)/);
});

test('footer exposes social share links', function () {
  var footer = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/footer.njk'), 'utf8');

  assert.match(footer, /aria-label="Share this site"/);
  assert.match(footer, /twitter\.com\/intent\/tweet/);
  assert.match(footer, /facebook\.com\/sharer\/sharer\.php/);
  assert.match(footer, /linkedin\.com\/sharing\/share-offsite/);
  assert.match(footer, /wa\.me\/\?text=/);
});
