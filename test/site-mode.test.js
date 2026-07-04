'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');
var vm = require('node:vm');

var repoRoot = path.join(__dirname, '..');
var modeSource = fs.readFileSync(path.join(repoRoot, 'src/assets/js/site-mode.js'), 'utf8');

function runMode(options) {
  var attributes = { 'data-site-mode': options.defaultMode || 'launch' };
  var stored = options.storedMode || '';
  var writes = [];
  var context = {
    URLSearchParams: URLSearchParams,
    document: {
      documentElement: {
        getAttribute: function (name) { return attributes[name] || null; },
        setAttribute: function (name, value) { attributes[name] = value; },
      },
    },
    window: {
      location: { search: options.search || '' },
      sessionStorage: {
        getItem: function () {
          if (options.storageError) throw new Error('storage unavailable');
          return stored;
        },
        setItem: function (key, value) {
          if (options.storageError) throw new Error('storage unavailable');
          writes.push([key, value]);
          stored = value;
        },
      },
    },
  };

  vm.runInNewContext(modeSource, context);
  return { mode: attributes['data-site-mode'], stored: stored, writes: writes };
}

test('site mode defaults to launch and restores a valid session mode', function () {
  assert.equal(runMode({}).mode, 'launch');
  assert.equal(runMode({ storedMode: 'full' }).mode, 'full');
});

test('valid query mode takes precedence and persists for the session', function () {
  var full = runMode({ search: '?oobCode=abc&site-mode=full', storedMode: 'launch' });
  assert.equal(full.mode, 'full');
  assert.equal(full.stored, 'full');
  assert.deepEqual(full.writes, [['ipace_site_mode', 'full']]);

  var launch = runMode({ search: '?site-mode=launch', storedMode: 'full' });
  assert.equal(launch.mode, 'launch');
  assert.equal(launch.stored, 'launch');
});

test('invalid query values are ignored without overwriting session state', function () {
  var result = runMode({ search: '?site-mode=preview', storedMode: 'full' });
  assert.equal(result.mode, 'full');
  assert.deepEqual(result.writes, []);
});

test('site mode still follows the query when session storage is unavailable', function () {
  assert.equal(runMode({ search: '?site-mode=full', storageError: true }).mode, 'full');
  assert.equal(runMode({ storageError: true }).mode, 'launch');
});

test('launch navigation and complete routes are marked declaratively', function () {
  var site = JSON.parse(fs.readFileSync(path.join(repoRoot, 'src/_data/site.json'), 'utf8'));
  var navigation = JSON.parse(fs.readFileSync(path.join(repoRoot, 'src/_data/navigation.json'), 'utf8'));
  var base = fs.readFileSync(path.join(repoRoot, 'src/_includes/layouts/base.njk'), 'utf8');
  var css = fs.readFileSync(path.join(repoRoot, 'src/assets/css/site.css'), 'utf8');

  assert.equal(site.defaultMode, 'launch');
  assert.deepEqual(
    navigation.public.filter(function (item) { return item.launchVisible === false; }).map(function (item) { return item.label; }),
    ['Evidence', 'Methodology', 'Updates']
  );
  assert.match(base, /data-site-mode="\{\{ site\.defaultMode or 'launch' \}\}"/);
  assert.match(base, /script src="\/assets\/js\/site-mode\.js"/);
  assert.match(base, /This section is not part of the public launch/);
  assert.match(css, /\[data-site-mode="launch"\] \[data-site-mode-only="full"\]/);
});
