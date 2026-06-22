const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');

test('member UI appends authenticated SoH readings to owned vehicles', function () {
  const memberAuth = read('src/assets/js/member-auth.js');
  const firebase = read('firebase.json');

  assert.match(memberAuth, /data-soh-update-form/);
  assert.match(memberAuth, /fetchWithIdentity\('\/api\/submit-soh'/);
  assert.match(memberAuth, /name=\"vehicleId\"/);
  assert.match(memberAuth, /State of Health history/);
  assert.match(firebase, /"source": "\/api\/submit-soh"/);
  assert.match(firebase, /"functionId": "SubmitSOH"/);
});

test('homepage and evidence dashboard load real public aggregate statistics', function () {
  const home = read('src/index.njk');
  const dashboard = read('src/evidence-dashboard.njk');
  const stats = read('src/assets/js/public-stats.js');

  assert.match(home, /data-public-stat="vehiclesRegistered"/);
  assert.match(dashboard, /data-public-stat="averageReportedSoh"/);
  assert.match(dashboard, /data-public-distribution="soh"/);
  assert.doesNotMatch(dashboard, /Illustrative data|Sample data|Placeholder data/);
  assert.match(stats, /fetch\('\/api\/public-stats'\)/);
});
