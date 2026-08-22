const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const test = require('node:test');

const read = (path) => readFileSync(path, 'utf8');

test('member dashboard uses a full-width tabbed vehicle workspace', function () {
  const dashboard = read('src/member/dashboard.njk');
  const script = read('src/assets/js/member-dashboard.js');

  assert.match(dashboard, /data-vehicle-workspace/);
  assert.match(dashboard, /role="tablist"/);
  assert.doesNotMatch(dashboard, /<div class="grid">/);
  assert.match(script, /role="tab"/);
  assert.match(script, /State of Health history/);
  assert.match(script, /<svg[^>]+role="img"/);
  assert.match(script, /Service events and faults/);
  assert.match(script, /<option value="fault" selected>Fault<\/option>/);
  assert.match(script, /Related campaigns or recalls/);
  assert.match(script, /value="H441"/);
  assert.match(script, /value="H448"/);
  assert.match(script, /value="H570"/);
  assert.match(script, /value="H571"/);
  assert.match(script, /value="H572"/);
  assert.match(script, /campaign-selector/);
  assert.match(script, /Search by provider name or postcode/);
  assert.match(script, /Authorised Jaguar Land Rover service provider/);
  assert.match(script, /Calculated automatically when both dates are entered/);
  assert.doesNotMatch(script, /name="daysToFinalFix"/);
  assert.match(script, /Courtesy vehicle offered/);
  assert.match(script, /Courtesy vehicle provided/);
  assert.match(script, /4 months or more/);
  assert.match(script, /Miles driven whilst faulty/);
  assert.match(script, /Goodwill payment received/);
  assert.match(script, /Warranty cover in place/);
  assert.match(script, /Responsibility or warranty dispute/);
  assert.match(script, /payload\[key\]\.push\(value\)/);
  assert.match(script, /data-not-future/);
  assert.match(script, /Measurement date cannot be in the future/);
  assert.match(script, /Event date cannot be in the future/);
  assert.match(script, /function validateNotFutureDates/);
  assert.match(script, /jaguar-uk-service-providers\.json/);
});

test('member dashboard supports owned SoH and service record editing and soft deletion', function () {
  var dashboard = read('src/assets/js/member-dashboard.js');
  assert.match(dashboard, /data-edit-soh/);
  assert.match(dashboard, /data-delete-record/);
  assert.match(dashboard, /\/api\/delete-soh/);
  assert.match(dashboard, /\/api\/delete-service-event/);
  assert.match(dashboard, /Type DELETE to confirm/);
});

test('service event editing is wired through the protected API', function () {
  const script = read('src/assets/js/member-dashboard.js');
  const auth = read('src/assets/js/member-auth.js');
  const firebase = read('firebase.json');
  const layout = read('src/_includes/layouts/base.njk');

  assert.match(script, /fetch\('\/api\/upsert-service-event'/);
  assert.match(script, /ipaceGetIdentityToken/);
  assert.match(auth, /new CustomEvent\('member:data'/);
  assert.match(firebase, /"source": "\/api\/\*\*"/);
  assert.match(firebase, /"functionId": "Api"/);
  assert.match(layout, /assets\/js\/member-dashboard\.js/);
});
