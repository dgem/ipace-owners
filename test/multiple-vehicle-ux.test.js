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
  assert.match(account, /href="\/member\/dashboard\/"[\s\S]*My Data/);
  assert.match(account, /account-layout__wide/);
  assert.match(memberAuth, /account-vehicle-grid/);
  assert.match(memberAuth, /Manage history/);
  assert.match(dashboard, /Your vehicles/);
  assert.match(dashboard, /Add vehicle/);
  assert.match(memberAuth, /Add your first vehicle/);
});

test('vehicle basics form invites adding each I-PACE separately', function () {
  var vehicleForm = fs.readFileSync(path.join(repoRoot, 'src/submit-vehicle-data.njk'), 'utf8');
  var identity = fs.readFileSync(path.join(repoRoot, 'src/assets/js/identity.js'), 'utf8');
  var multistep = fs.readFileSync(path.join(repoRoot, 'src/assets/js/multistep-form.js'), 'utf8');

  assert.match(vehicleForm, /If you own, owned, or help with more than one I-PACE/);
  assert.match(vehicleForm, /Add one vehicle at a time/);
  assert.match(vehicleForm, /Add another vehicle/);
  assert.match(vehicleForm, /Vehicle data starts here/);
  assert.match(vehicleForm, /use My Data to add further SoH readings/);
  assert.doesNotMatch(vehicleForm, /It does not yet collect recall, repair, loan car, payment/);
  assert.match(vehicleForm, /data-require-one="vin registration"/);
  assert.match(vehicleForm, /Provide either the VIN or registration before continuing/);
  assert.match(vehicleForm, /data-database-result-icon/);
  assert.match(vehicleForm, /data-database-error-message/);
  assert.match(vehicleForm, /data-database-success-actions hidden/);
  assert.match(vehicleForm, /data-database-error-actions hidden/);
  assert.match(multistep, /function validateRequiredGroups/);
  assert.match(multistep, /data-require-one/);
  assert.match(identity, /setResultIcon/);
  assert.match(identity, /setDatabaseErrorMessage/);
  assert.match(identity, /showDatabaseError\(result, err && err\.message/);
});
