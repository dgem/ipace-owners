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

test('account notices share a responsive two-column group with consistent links', () => {
  const account = read('src/member/account.njk');
  const surveys = read('src/assets/js/surveys.js');
  const css = read('src/assets/css/site.css');

  assert.match(account, /class="member-account-notices"/);
  assert.match(css, /\.member-account-notices[\s\S]*grid-template-columns:\s*repeat\(2, minmax\(0, 1fr\)\)/);
  assert.match(surveys, /historyAction[\s\S]*btn btn--primary btn--sm/);
  assert.doesNotMatch(surveys, /btn btn--secondary btn--sm" href="\/member\/surveys\/\?filter=closed">Past surveys/);
  assert.match(account, /href="\/member\/surveys\/" class="btn btn--primary btn--sm"/);
  assert.match(account, /href="\/member\/stories\/" class="btn btn--primary btn--sm"/);
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

test('member experience editor inserts uploaded images instead of arbitrary links', () => {
  const stories = read('src/assets/js/member-stories.js');
  const publicStories = read('src/assets/js/public-member-stories.js');

  assert.match(stories, /data-insert-story-image/);
  assert.match(stories, /data-insert-story-media/);
  assert.match(stories, /data-insert-pending-media/);
  assert.match(stories, /attachUploadedMedia/);
  assert.match(stories, /data-editor-value="blockquote"/);
  assert.match(stories, /tag === 'blockquote'/);
  assert.match(stories, /story-media:/);
  assert.match(stories, /data-story-media-id/);
  assert.match(stories, /story.status === 'submitted'/);
  assert.match(stories, /story.analysis && story.analysis.sentiment/);
  assert.doesNotMatch(stories, /window\.prompt\('Paste a web link/);
  assert.doesNotMatch(stories, /data-editor-link/);
  assert.match(publicStories, /story-inline-image/);
  assert.match(publicStories, /pathname.split\('\/'\)/);
  assert.match(publicStories, /member-stories\/'\s*\+\s*encodeURIComponent/);
  assert.match(publicStories, /<video src=/);
  assert.match(publicStories, /<blockquote>/);
});

test('member and admin story lists link to separate editors and new story IDs are retry-stable', () => {
  const listPage = read('src/member/stories.njk');
  const editorPage = read('src/member/stories/edit.njk');
  const memberJS = read('src/assets/js/member-stories.js');
  const adminListJS = read('src/assets/js/admin-member-stories.js');
  const adminEditorJS = read('src/assets/js/admin-member-story-edit.js');
  const adminEditorPage = read('src/admin/member-stories/edit.njk');
  const styles = read('src/assets/css/site.css');

  assert.match(listPage, /data-view="list"/);
  assert.match(editorPage, /data-view="editor"/);
  assert.match(memberJS, /member\/stories\/edit\/\?id=/);
  assert.match(memberJS, /crypto\.randomUUID/);
  assert.match(memberJS, /if \(saving\) return/);
  assert.match(memberJS, /class="btn btn--secondary btn--sm" type="button" data-insert-pending-media/);
  assert.match(memberJS, /class="btn btn--danger btn--sm" type="button" data-remove-media/);
  assert.match(adminListJS, /admin-story-row__preview/);
  assert.match(adminEditorPage, /data-admin-story-editor/);
  assert.match(adminEditorJS, /Story \(Markdown source\)/);
  assert.doesNotMatch(styles, /\.story-media-card \.btn--danger\s*\{\s*background:\s*var\(--color-accent\)/);
});

test('member story upload errors retain a support correlation reference', () => {
  const stories = read('src/assets/js/member-stories.js');

  assert.match(stories, /error\.traceId = data\.traceId/);
  assert.match(stories, /reference ' \+ error\.traceId/);
});

test('member story media preflights existing usage and resizes large browser images', () => {
  const stories = read('src/assets/js/member-stories.js');

  assert.match(stories, /existingMediaUsage\.bytes \+ bytes/);
  assert.match(stories, /createImageBitmap/);
  assert.match(stories, /imageResizePixels/);
});
