'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var css = fs.readFileSync(path.join(__dirname, '..', 'src', 'assets', 'css', 'site.css'), 'utf8');
var baseLayout = fs.readFileSync(path.join(__dirname, '..', 'src', '_includes', 'layouts', 'base.njk'), 'utf8');

test('mobile launch layout prevents horizontal clipping and keeps cookie notice compact', function () {
  assert.match(css, /html\s*\{[\s\S]*overflow-x: clip;/);
  assert.match(css, /body\s*\{[\s\S]*overflow-x: clip;/);
  assert.match(css, /\.hero__inner\s*\{[\s\S]*min-width: 0;/);
  assert.match(css, /\.hero__content\s*\{[\s\S]*width: 100%;[\s\S]*min-width: 0;/);
  assert.match(css, /\.hero__media\s*\{[\s\S]*max-width: 100%;[\s\S]*min-width: 0;/);
  assert.match(css, /@media \(max-width: 39\.99em\)\s*\{[\s\S]*\.site-header \.identity-controls\s*\{[\s\S]*display: none;/);
  assert.match(css, /@media \(max-width: 39\.99em\)\s*\{[\s\S]*\.cookie-notice\s*\{[\s\S]*max-height: 34vh;/);
  assert.match(baseLayout, /We use essential cookies only for sign-in and to remember this notice/);
});
