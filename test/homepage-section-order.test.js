const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const homepage = fs.readFileSync(path.join(__dirname, '..', 'src', 'index.njk'), 'utf8');

test('launch homepage leads with recruitment and constructive resolution', function () {
  const launchEnd = homepage.indexOf('<div data-site-mode-only="full">');
  const launchHomepage = homepage.slice(0, launchEnd);

  assert.match(launchHomepage, /I-PACE owners working together for fair outcomes/);
  assert.match(launchHomepage, /traction battery faults and recall uncertainty/);
  assert.match(launchHomepage, /shape a fair offer that works for as many UK owners\s+as possible/);
  assert.match(launchHomepage, /Why now\?/);
  assert.match(launchHomepage, /Battery issues and recalls need an organised response/);
  assert.match(launchHomepage, /warranty uncertainty, and inconsistent support/);
  assert.match(launchHomepage, /Owners elsewhere can still register interest/);
  assert.match(launchHomepage, /before Jaguar's next vehicle launch/);
  assert.match(launchHomepage, /not another forum/);
  assert.match(launchHomepage, /not building an open comment\s+board/);
  assert.match(launchHomepage, /legal counsel within the group\s+to help test the process and options/);
  assert.match(launchHomepage, /does not enrol you in legal\s+action/);
  assert.equal((launchHomepage.match(/href="\/join\/"/g) || []).length, 2);
  assert.doesNotMatch(launchHomepage, /H447|H570|State of Health/);
});

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
  assert.match(homepage, /H447 \/ H570 \/ H571 \/ H572/);
  assert.ok(
    homepage.indexOf('Get started', participation) <
      homepage.indexOf('This is not a legal action', participation)
  );
});
