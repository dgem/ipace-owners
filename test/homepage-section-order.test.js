const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const homepage = fs.readFileSync(path.join(__dirname, '..', 'src', 'index.njk'), 'utf8');

function launchHomepage() {
  const launchEnd = homepage.indexOf('<div data-site-mode-only="full">');
  return homepage.slice(0, launchEnd);
}

test('launch homepage leads with recruitment and constructive resolution', function () {
  const launch = launchHomepage();

  assert.match(launch, /I-PACE owners working together for fair outcomes/);
  assert.match(launch, /Register your support around I-PACE battery and recall issues/);
  assert.match(launch, /Traction battery faults/);
  assert.match(launch, /H447 \/ H570 \/ H571 \/ H572 campaigns/);
  assert.match(launch, /Approaching the 8-year \/ 100,000-mile battery warranty/);
  assert.match(launch, /Inconsistent owner experiences/);
  assert.match(launch, /Why now\?/);
  assert.match(launch, /Recalls and traction battery failures need an organised response/);
  assert.match(launch, /before Jaguar's next vehicle launch/);
  assert.match(launch, /not another forum/);
  assert.match(launch, /not building an open comment\s+board/);
  assert.match(launch, /legal counsel within the group\s+to help test the process and options/);
  assert.match(launch, /does not enrol you in legal\s+action/);
  assert.match(launch, /only asking owners to register with a name\s+and email address/);
  assert.equal((launch.match(/href="\/join\/"/g) || []).length, 2);
  assert.doesNotMatch(launch, /State of Health|Submit Vehicle Data/);
});

test('launch homepage uses the structured Why now design, not the full-mode section', function () {
  const launch = launchHomepage();
  const whyStart = launch.indexOf('<!-- Launch Why now? -->');
  const routeStart = launch.indexOf('<section class="section bg-muted">', whyStart);
  const launchWhy = launch.slice(whyStart, routeStart);

  assert.ok(whyStart > -1);
  assert.ok(routeStart > whyStart);
  assert.match(launchWhy, /class="text-muted why-now__intro"/);
  assert.match(launchWhy, /class="two-column launch-why-grid"/);
  assert.equal((launchWhy.match(/class="why-item"/g) || []).length, 4);
  assert.match(launchWhy, /class="callout callout--info launch-registration-callout"/);
  assert.doesNotMatch(launchWhy, /State of Health|Submit Vehicle Data/);
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
