'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('header exposes one My Data action and account via user email when signed in', function () {
  var header = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/header.njk'), 'utf8');
  var mobileNav = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/mobile-nav.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');

  assert.match(header, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.equal((header.match(/>My Data<\/a>/g) || []).length, 1);
  assert.equal((header.match(/>Dashboard<\/a>/g) || []).length, 0);
  assert.match(header, /id="identity-user-display" class="identity-controls__user btn btn--sm btn--ghost" href="\/member\/account\/"/);
  assert.doesNotMatch(header, />Account<\/a>/);
  assert.match(mobileNav, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.equal((mobileNav.match(/>My Data<\/a>/g) || []).length, 1);
  assert.equal((mobileNav.match(/>Dashboard<\/a>/g) || []).length, 0);
  assert.match(mobileNav, /href="\/member\/account\/"[\s\S]*data-requires-auth[\s\S]*>My account<\/a>/);
  assert.match(identityJs, /setVisibility\('\[data-requires-auth\]', true\)/);
  assert.match(identityJs, /setVisibility\('\[data-requires-guest\]', false\)/);
  assert.match(identityJs, /setAttribute\('aria-label', 'My account'\)/);
});

test('admin tools are discoverable only when Firebase token claims permit them', function () {
  var header = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/header.njk'), 'utf8');
  var mobileNav = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/mobile-nav.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');
  var css = fs.readFileSync(path.join(repoRoot, 'src/assets/css/site.css'), 'utf8');

  assert.match(header, /site-admin-nav[\s\S]*data-requires-admin[\s\S]*navigation\.admin/);
  assert.match(mobileNav, /mobile-nav__admin[\s\S]*data-requires-admin[\s\S]*navigation\.admin/);
  assert.doesNotMatch(header, /data-requires-admin[^>]*>Admin<\/a>/);
  assert.match(identityJs, /user\.getIdTokenResult\(\)/);
  assert.match(identityJs, /claims\.admin === true \|\| roles\.indexOf\('admin'\) !== -1/);
  assert.match(identityJs, /setVisibility\('\[data-requires-admin\]', false\)/);
  assert.match(identityJs, /setVisibility\('\[data-requires-admin\]', isAdmin\)/);
  assert.match(css, /@media \(min-width: 64em\)[\s\S]*\.site-admin-nav\s*\{[\s\S]*display: flex;/);
});

test('footer exposes social share links', function () {
  var footer = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/footer.njk'), 'utf8');

  assert.match(footer, /aria-label="Share this site"/);
  assert.match(footer, /twitter\.com\/intent\/tweet/);
  assert.match(footer, /facebook\.com\/sharer\/sharer\.php/);
  assert.match(footer, /linkedin\.com\/sharing\/share-offsite/);
  assert.match(footer, /wa\.me\/\?text=/);
});

test('footer links to the public source repository with a GitHub icon', function () {
  var footer = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/footer.njk'), 'utf8');

  assert.match(footer, /href="https:\/\/github\.com\/dgem\/ipace-owners"/);
  assert.match(footer, /class="site-footer__github-link"/);
  assert.match(footer, /<svg[^>]+aria-hidden="true"/);
  assert.match(footer, /<span>GitHub<\/span>/);
});
