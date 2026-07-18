const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..');

test('member account and vehicle forms live under the member route namespace', function () {
  assert.equal(fs.existsSync(path.join(root, 'src/member/account.njk')), true);
  assert.equal(fs.existsSync(path.join(root, 'src/member/submit-vehicle-data.njk')), true);
  assert.equal(fs.existsSync(path.join(root, 'src/account.njk')), false);
  assert.equal(fs.existsSync(path.join(root, 'src/submit-vehicle-data.njk')), false);
});

test('legacy account and vehicle form routes redirect permanently', function () {
  const firebase = JSON.parse(fs.readFileSync(path.join(root, 'firebase.json'), 'utf8'));
  const redirects = firebase.hosting.redirects;

  [
    ['/account', '/member/account/'],
    ['/account/**', '/member/account/'],
    ['/submit-vehicle-data', '/member/submit-vehicle-data/'],
    ['/submit-vehicle-data/**', '/member/submit-vehicle-data/'],
  ].forEach(function ([source, destination]) {
    assert.ok(
      redirects.some(function (redirect) {
        return redirect.source === source
          && redirect.destination === destination
          && redirect.type === 301;
      }),
      source + ' should redirect permanently to ' + destination,
    );
  });
});
