const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.resolve(__dirname, '..');
const read = (name) => fs.readFileSync(path.join(root, name), 'utf8');

test('member experience entry points share the server-resolved feature gate', () => {
  const callout = read('src/_includes/partials/member-experience-callout.njk');
  const account = read('src/member/account.njk');
  const stories = read('src/member/stories.njk');
  const dashboard = read('src/assets/js/member-dashboard.js');
  const navigation = JSON.parse(read('src/_data/navigation.json'));

  assert.match(callout, /data-member-feature="member-experiences"/);
  assert.match(account, /data-member-feature="member-experiences"/);
  assert.match(stories, /data-member-feature-denied="member-experiences"/);
  assert.match(dashboard, /data-member-feature="member-experiences"/);
  assert.equal(navigation.member.find((item) => item.url === '/member/stories/').feature, 'member-experiences');
});

test('member calls to action and surveys have reusable versioned dismissal identities', () => {
  const callout = read('src/_includes/partials/member-experience-callout.njk');
  const surveys = read('src/assets/js/surveys.js');
  const layout = read('src/_includes/layouts/base.njk');

  assert.match(callout, /data-notification-id="cta:member-experiences:v1"/);
  assert.match(surveys, /survey:open:/);
  assert.match(surveys, /survey:upcoming:/);
  assert.match(layout, /member-notifications\.js/);
  assert.match(layout, /member-features\.js/);
});

test('deployment passes member feature rules to functions', () => {
  const moduleActions = read('infra/opentofu/modules/ipace-owners/github-actions.tf');
  const staging = read('infra/opentofu/env/staging.tfvars.example');
  const production = read('infra/opentofu/env/production.tfvars.example');
  const stagingWorkflow = read('.github/workflows/gcp-firebase-staging.yml');
  const productionWorkflow = read('.github/workflows/gcp-firebase-production.yml');

  assert.match(moduleActions, /MEMBER_FEATURE_ACCESS_/);
  assert.match(staging, /member-experiences\s*=\s*"all"/);
  assert.match(production, /member-experiences\s*=\s*"admin,member-experiences-preview"/);
  assert.match(stagingWorkflow, /MEMBER_FEATURE_ACCESS_STAGING[\s\S]*member-experiences.*all/);
  assert.match(productionWorkflow, /MEMBER_FEATURE_ACCESS_PRODUCTION[\s\S]*member-experiences.*admin,member-experiences-preview/);
});

test('member experiences use the membership display name', () => {
  const stories = read('src/assets/js/member-stories.js');
  const preferences = read('src/assets/js/member-auth.js');

  assert.match(stories, /Published as <strong>/);
  assert.match(stories, /state\.displayName/);
  assert.doesNotMatch(stories, /name="displayName"/);
  assert.match(preferences, /payload\.displayName/);
});
