const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const homepage = fs.readFileSync(path.join(__dirname, '..', 'src', 'index.njk'), 'utf8');

test('homepage orders Why now, features, and participation actions', function () {
  const whyNow = homepage.indexOf('<!-- Why now? -->');
  const features = homepage.indexOf('<!-- Feature cards -->');
  const participation = homepage.indexOf('<!-- Participation and next steps -->');

  assert.ok(whyNow > homepage.indexOf('<!-- Hero -->'));
  assert.ok(features > whyNow);
  assert.ok(participation > features);
  assert.match(homepage.slice(whyNow, features), /class="text-muted why-now__intro"/);
  assert.match(homepage.slice(whyNow, features), /class="two-column"/);
  assert.match(homepage.slice(participation), /This is not a legal action/);
  assert.match(homepage.slice(participation), /Get started/);
});
