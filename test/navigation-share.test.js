'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('header exposes one My Data account action without exposing the member email', function () {
  var header = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/header.njk'), 'utf8');
  var mobileNav = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/mobile-nav.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');
  var css = fs.readFileSync(path.join(repoRoot, 'src/assets/css/site.css'), 'utf8');

  assert.match(header, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.equal((header.match(/>My Data<\/a>/g) || []).length, 1);
  assert.equal((header.match(/>Dashboard<\/a>/g) || []).length, 0);
  assert.match(header, /href="\/member\/account\/" class="btn btn--sm btn--primary" data-requires-auth[^>]*>My Data<\/a>/);
  assert.doesNotMatch(header, /identity-user-display/);
  assert.match(header, /id="identity-mobile-header-login-btn"[^>]+href="\/member\/account\/"[^>]*>Sign in<\/a>/);
  assert.match(header, /aria-label="Member breadcrumb"/);
  assert.match(header, /site-context-nav__current" aria-current="page">\{\{ memberBreadcrumb \}\}/);
  assert.match(header, /aria-label="Admin breadcrumb"/);
  assert.match(header, /site-context-nav__current" aria-current="page">\{\{ adminBreadcrumb \}\}/);
  assert.match(mobileNav, /data-requires-guest[\s\S]*>\{\{ item\.label \}\}<\/a>/);
  assert.match(mobileNav, /\{% for item in navigation\.member %\}/);
  assert.equal((mobileNav.match(/>Dashboard<\/a>/g) || []).length, 0);
  assert.match(mobileNav, /mobile-nav__identity-label">Member/);
  assert.match(mobileNav, /id="identity-mobile-login-btn"[\s\S]*>Sign in<\/a>/);
  assert.match(identityJs, /mobileLoginBtn\.style\.display = 'none'/);
  assert.match(identityJs, /mobileLoginBtn\.style\.display = ''/);
  assert.match(identityJs, /setVisibility\('\[data-requires-auth\]', true\)/);
  assert.match(identityJs, /setVisibility\('\[data-requires-guest\]', false\)/);
  assert.match(identityJs, /mobileHeaderLoginBtn\.style\.display = 'none'/);
  assert.match(identityJs, /mobileHeaderLoginBtn\.style\.display = ''/);
  assert.match(css, /\.site-header \.mobile-header-login\s*\{\s*display: none;/);
  assert.match(css, /@media \(max-width: 39\.99em\)[\s\S]*\.site-header \.mobile-header-login\s*\{[\s\S]*display: inline-flex;/);
  assert.match(css, /\.site-context-nav__current\[aria-current="page"\][\s\S]*font-weight: 800;/);
  assert.match(css, /\.site-context-nav__list\s*\{[\s\S]*list-style: none !important;/);
  assert.match(css, /\.mobile-nav__identity \.btn--secondary\s*\{[\s\S]*color: #ffffff;/);
  assert.doesNotMatch(identityJs, /identity-user-display|userDisplay/);
});

test('every admin page uses the compact Admin breadcrumb pattern', function () {
  var pages = {
    'index.njk': 'Dashboard',
    'review-queue.njk': 'Review queue',
    'outreach.njk': 'Facebook outreach',
    'email-campaigns.njk': 'Email campaigns',
    'instagram-campaigns.njk': 'Instagram campaigns',
    'surveys.njk': 'Surveys'
  };

  Object.entries(pages).forEach(function (entry) {
    var page = fs.readFileSync(path.join(repoRoot, 'src/admin', entry[0]), 'utf8');
    assert.match(page, /adminNavigation: true/);
    assert.match(page, new RegExp('adminBreadcrumb: ' + entry[1]));
    assert.match(page, /page-header page-header--compact/);
    assert.doesNotMatch(page, /page-header__eyebrow">Admin/);
  });
});

test('one Admin destination is discoverable only when Firebase token claims permit it', function () {
  var header = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/header.njk'), 'utf8');
  var mobileNav = fs.readFileSync(path.join(repoRoot, 'src/_includes/partials/mobile-nav.njk'), 'utf8');
  var identityJs = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');
  var navigation = fs.readFileSync(path.join(repoRoot, 'src/_data/navigation.json'), 'utf8');

  assert.match(header, /href="\/admin\/" class="btn btn--sm btn--ghost" data-requires-admin[^>]*>Admin<\/a>/);
  assert.match(mobileNav, /href="\/admin\/" class="btn btn--sm btn--secondary full-width"[^>]*data-requires-admin[^>]*>Admin<\/a>/);
  assert.doesNotMatch(header, /site-admin-nav|navigation\.admin/);
  assert.doesNotMatch(mobileNav, /mobile-nav__admin|navigation\.admin/);
  assert.doesNotMatch(navigation, /"admin"\s*:/);
  assert.match(identityJs, /user\.getIdTokenResult\(\)/);
  assert.match(identityJs, /claims\.admin === true \|\| roles\.indexOf\('admin'\) !== -1/);
  assert.match(identityJs, /setVisibility\('\[data-requires-admin\]', false\)/);
  assert.match(identityJs, /setVisibility\('\[data-requires-admin\]', isAdmin\)/);
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
