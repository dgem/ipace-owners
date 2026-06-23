'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var repoRoot = path.join(__dirname, '..');

test('member/account UI treats vehicle records as a list', function () {
  var account = fs.readFileSync(path.join(repoRoot, 'src/account.njk'), 'utf8');
  var dashboard = fs.readFileSync(path.join(repoRoot, 'src/member/dashboard.njk'), 'utf8');
  var memberAuth = fs.readFileSync(path.join(repoRoot, 'src/assets/js/member-auth.js'), 'utf8');

  assert.match(account, /Your vehicles/);
  assert.match(account, /Add another vehicle/);
  assert.match(account, /href="\/member\/dashboard\/"[\s\S]*Member dashboard/);
  assert.match(account, /account-layout__wide/);
  assert.match(memberAuth, /account-vehicle-grid/);
  assert.match(memberAuth, /Manage history/);
  assert.match(dashboard, /Your vehicles/);
  assert.match(dashboard, /Add vehicle/);
  assert.match(memberAuth, /Add your first vehicle/);
});

test('vehicle basics form invites adding each I-PACE separately', function () {
  var vehicleForm = fs.readFileSync(path.join(repoRoot, 'src/submit-vehicle-data.njk'), 'utf8');

  assert.match(vehicleForm, /If you own, owned, or help with more than one I-PACE/);
  assert.match(vehicleForm, /Add one vehicle at a time/);
  assert.match(vehicleForm, /Add another vehicle/);
});
