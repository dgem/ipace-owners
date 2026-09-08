const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const reminderPage = fs.readFileSync(path.join(root, 'src/admin/email-campaigns.njk'), 'utf8');
const reminderScript = fs.readFileSync(path.join(root, 'src/assets/js/email-campaigns.js'), 'utf8');
const marketingPage = fs.readFileSync(path.join(root, 'src/admin/marketing-messages.njk'), 'utf8');
const marketingScript = fs.readFileSync(path.join(root, 'src/assets/js/marketing-messages.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');
const dashboard = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');

test('registration reminders retain only the transactional fresh-link workflow', function () {
  assert.match(reminderPage, /data-admin-container/);
  assert.match(reminderPage, /Send registration reminders/);
  assert.match(reminderPage, /fresh private Firebase sign-in link/);
  assert.match(reminderPage, /\/admin\/marketing-messages\//);
  assert.match(reminderPage, /\/api\/admin\/reengagement-preview/);
  assert.match(reminderPage, /\/api\/admin\/reengagement-send/);
  assert.doesNotMatch(reminderPage, /September Survey/);
  assert.doesNotMatch(reminderPage, /Freeform/);
  assert.doesNotMatch(reminderPage, /data-campaign-tabs/);
  assert.match(layout, /emailCampaigns[\s\S]*email-campaigns\.js/);
  assert.match(reminderScript, /getIdToken\(\)/);
  assert.match(reminderScript, /expectedEligible: current\.eligible/);
});

test('marketing messages provides resumable prepared campaigns', function () {
  for (const template of [
    'survey-september-2026',
    'jlr-contact',
    'find-members',
    'reach-1000'
  ]) {
    assert.match(marketingPage, new RegExp('value="' + template + '"'));
  }
  assert.match(marketingPage, /September survey invitation/);
  assert.match(marketingPage, /source-controlled copy/);
  assert.match(marketingScript, /\/api\/admin\/marketing-message-templates/);
  assert.match(marketingScript, /templateId: templateID\.value/);
  assert.match(marketingScript, /campaignId: campaignID\.value/);
  assert.match(marketingPage, /batches of up to 100 emails/);
  assert.match(marketingScript, /\/api\/admin\/marketing-message-deliveries/);
  assert.match(marketingScript, /templateId: templateID\.value/);
  assert.match(marketingScript, /already sent on/);
  assert.match(marketingScript, /Batch complete:/);
  assert.match(marketingPage, /Campaign delivery history/);
  assert.match(marketingScript, /sendButton\.disabled = true/);
  assert.match(marketingPage, /Editing the copied content/);
  assert.match(marketingScript, /getIdToken\(\)/);
  assert.match(marketingScript, /data\.eligible/);
  assert.doesNotMatch(marketingScript, /innerHTML\s*=/);
});

test('admin dashboard distinguishes marketing broadcasts from registration reminders', function () {
  assert.match(dashboard, />Registration reminders<\/h3>/);
  assert.match(dashboard, />Registration Reminders<\/a>/);
  assert.match(dashboard, /September survey/);
  assert.match(dashboard, />Marketing Messages<\/a>/);
});
