const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const read = (file) => fs.readFileSync(path.join(root, file), 'utf8');

test('member UI appends authenticated SoH readings to owned vehicles', function () {
  const memberAuth = read('src/assets/js/member-auth.js');
  const memberDashboard = read('src/assets/js/member-dashboard.js');
  const firebase = read('firebase.json');

  assert.match(memberDashboard, /data-soh-update-form/);
  assert.match(memberDashboard, /name="vehicleId"/);
  assert.match(memberDashboard, /State of Health history/);
  assert.match(memberAuth, /fetchWithIdentity\('\/api\/submit-soh'/);
  assert.match(firebase, /"source": "\/api\/\*\*"/);
  assert.match(firebase, /"functionId": "Api"/);
  assert.doesNotMatch(firebase, /"functionId": "SubmitSOH"/);
});

test('homepage and evidence dashboard load real public aggregate statistics', function () {
  const home = read('src/index.njk');
  const dashboard = read('src/evidence-dashboard.njk');
  const wreath = read('src/_includes/partials/racing-wreath.njk');
  const stats = read('src/assets/js/public-stats.js');

  assert.match(home, /racingWreath\("owners-joined-wreath", "joinedOwners", "Owners joined", "17th July 2026", "2026-07-17"\)/);
  assert.match(home, /racingWreath\("cars-registered-wreath", "vehiclesRegistered"/);
  assert.match(home, /racingWreath\("soh-readings-wreath", "sohReadings"/);
  assert.match(home, /racingWreath\("service-records-wreath", "serviceEventsLogged"/);
  assert.match(wreath, /launch-member-count__date/);
  assert.match(wreath, /data-public-stat="\{\{ statKey \}\}"/);
  assert.match(wreath, /Since <time datetime="\{\{ datetime \}\}">\{\{ note \}\}<\/time>/);
  assert.match(dashboard, /data-public-stat="averageReportedSoh"/);
  assert.match(dashboard, /data-public-stat="serviceEventsLogged"/);
  assert.match(dashboard, /data-public-distribution="soh"/);
  assert.doesNotMatch(dashboard, /Illustrative data|Sample data|Placeholder data/);
  assert.match(stats, /fetch\('\/api\/public-stats\?v=6'\)/);
  assert.match(stats, /displayedCharacters = count\.toLocaleString\('en-GB'\)\.length/);
  assert.match(stats, /displayedCharacters >= 6 \? 'large' : displayedCharacters >= 5 \? 'five' : displayedCharacters >= 4 \? 'four'/);

  const css = read('src/assets/css/site.css');
  assert.match(css, /data-count-size="three"[^}]*font-size: 1\.75rem/s);
  assert.match(css, /data-count-size="four"[^}]*font-size: 1\.5rem/s);
  assert.match(css, /data-count-size="five"[^}]*font-size: 1\.5rem/s);
  assert.match(css, /data-count-size="large"[^}]*font-size: 1\.25rem/s);
  assert.match(css, /\.launch-evidence-wreaths[^}]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\)/s);
});
