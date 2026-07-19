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
  assert.match(script, /value="H570"/);
  assert.match(script, /value="H571"/);
  assert.match(script, /value="H572"/);
  assert.match(script, /Days from fault to final fix/);
  assert.match(script, /Courtesy vehicle offered/);
  assert.match(script, /Courtesy vehicle provided/);
  assert.match(script, /Delay due to parts/);
  assert.match(script, /Warranty cover in place/);
  assert.match(script, /Responsibility or warranty dispute/);
  assert.match(script, /payload\[key\]\.push\(value\)/);
  assert.match(script, /data-not-future/);
  assert.match(script, /Measurement date cannot be in the future/);
  assert.match(script, /Event date cannot be in the future/);
  assert.match(script, /function validateNotFutureDates/);
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
