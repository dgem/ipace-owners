const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const page = fs.readFileSync(path.join(root, 'src/admin/email-campaigns.njk'), 'utf8');
const script = fs.readFileSync(path.join(root, 'src/assets/js/email-campaigns.js'), 'utf8');
const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');
const campaignBackend = fs.readFileSync(path.join(root, 'functions/firebase-go/custom_email_campaigns.go'), 'utf8');
const specialisedCampaignBackend = fs.readFileSync(path.join(root, 'functions/firebase-go/email_campaigns.go'), 'utf8');

test('admin email campaign page is gated and describes bounded sending', function () {
  assert.match(page, /data-admin-container/);
  assert.match(page, /data-admin-content hidden/);
  assert.match(page, /without revealing addresses/i);
  assert.match(page, /batches of 100/i);
  assert.match(page, /Send registration reminder emails/);
  assert.match(page, /What each recipient will receive/);
  assert.doesNotMatch(page, /data-campaign-send hidden/);
  assert.match(page, /data-campaign-send-button disabled/);
  assert.match(layout, /emailCampaigns[\s\S]*email-campaigns\.js/);
  assert.doesNotMatch(page, /Admin navigation/);
});

test('the claim-gated header links once to the admin dashboard', function () {
  const outreach = fs.readFileSync(path.join(root, 'src/admin/outreach.njk'), 'utf8');
  const review = fs.readFileSync(path.join(root, 'src/admin/review-queue.njk'), 'utf8');
  const header = fs.readFileSync(path.join(root, 'src/_includes/partials/header.njk'), 'utf8');
  assert.doesNotMatch(outreach, /Admin navigation/);
  assert.doesNotMatch(review, /Admin navigation/);
  assert.match(header, /href="\/admin\/" class="btn btn--sm btn--ghost" data-requires-admin[^>]*>Admin<\/a>/);
  assert.doesNotMatch(header, /site-admin-nav|navigation\.admin/);
});

test('admin index is a gated dashboard of implemented tools', function () {
  const dashboard = fs.readFileSync(path.join(root, 'src/admin/index.njk'), 'utf8');
  assert.match(dashboard, /data-admin-container/);
  assert.match(dashboard, /data-admin-content hidden/);
  assert.match(dashboard, />Review Queue<\/a>/);
  assert.match(dashboard, />Facebook Assistant<\/a>/);
  assert.match(dashboard, />Email Campaigns<\/a>/);
  assert.match(dashboard, />Instagram Campaigns<\/a>/);
  assert.match(dashboard, />Member Surveys<\/a>/);
  assert.equal((dashboard.match(/class="admin-tool-logo"/g) || []).length, 5);
  assert.equal((dashboard.match(/class="btn btn--primary" href="\/admin\//g) || []).length, 5);
  assert.match(dashboard, /not linked prematurely/);
  assert.match(dashboard, /admin-dashboard-tools/);
  assert.match(dashboard, /admin-dashboard-insights/);
});

test('email campaign browser sends tokens and explicit confirmation data', function () {
  assert.match(script, /getIdToken\(\)/);
  assert.match(page, /\/api\/admin\/reengagement-preview/);
  assert.match(page, /\/api\/admin\/reengagement-send/);
  assert.match(page, /\/api\/admin\/member-referral-preview/);
  assert.match(page, /\/api\/admin\/member-referral-send/);
  assert.match(page, /\/api\/admin\/all-members-drive-preview/);
  assert.match(page, /\/api\/admin\/all-members-drive-send/);
  assert.match(page, /verified and unverified Join records/);
  assert.match(script, /expectedEligible: current\.eligible/);
  assert.match(script, /confirmation: confirmInput\.value/);
  assert.match(script, /emailHTML\.srcdoc = data\.emailPreview\.html/);
  assert.match(script, /emailText\.textContent = data\.emailPreview\.text/);
  assert.match(script, /emailPreview\.hidden = false/);
  assert.match(page, /data-campaign-email-html[^>]+sandbox/);
  assert.match(page, /View plain-text alternative/);
  assert.doesNotMatch(script, /recipient|emailAddress|\.email\b/i);
});

test('member referral campaign previews the exact HTML email in a readable sandbox', function () {
  const css = fs.readFileSync(path.join(root, 'src/assets/css/site.css'), 'utf8');
  assert.match(page, /Ask members to invite one more I-PACE owner/);
  assert.match(page, /registered members whose Join record includes contact consent/);
  assert.match(css, /\.email-preview__viewport iframe[\s\S]*height: 52rem/);
  assert.match(css, /\.email-preview pre[\s\S]*color: var\(--color-text\)/);
  assert.doesNotMatch(page, /data-campaign-share-preview/);
});

test('referral campaigns include a suggested stronger-together sharing CTA', function () {
  assert.match(specialisedCampaignBackend, /I-PACE owners are stronger together/);
  assert.match(specialisedCampaignBackend, /Own an I-PACE\\?/);
  assert.match(specialisedCampaignBackend, /help us reach 1,000 members/);
  assert.match(specialisedCampaignBackend, /quote=/);
  assert.ok(specialisedCampaignBackend.includes('https://wa.me/?text='));
});

test('custom campaign editor provides history, Markdown preview, reruns and safe substitutions', function () {
  for (const field of [
    'membersJoined',
    'membersVerified',
    'memberFirstName',
    'memberLastName',
    'memberTittle',
    'memberJoined',
    'memberVerified',
    'memberVehicles',
    'vehiclesRegisteredCount',
    'vehiclesSoHReadingsCount',
    'serviceFaultRecordsCount'
  ]) {
    assert.match(campaignBackend, new RegExp('"' + field + '"'));
  }
  assert.match(campaignBackend, /NewAggregationQuery\(\)\.\s*WithCount\("serviceFaultRecords"\)/);
  assert.doesNotMatch(campaignBackend, /serviceIter := db\.Collection\("serviceEvents"\)\.Documents/);
  assert.match(page, /Create a member email campaign/);
  assert.match(page, /href="#campaign-tools" data-campaign-open-tab="freeform"/);
  assert.match(page, /data-custom-campaign-markdown/);
  assert.match(page, /data-custom-campaign-email-html[^>]+sandbox/);
  assert.match(page, /Previous campaigns/);
  assert.match(page, /Tweak and rerun/);
  assert.ok(
    page.indexOf('data-email-campaign-history') < page.indexOf('data-custom-email-campaign'),
    'campaign history should appear before the new-campaign composer'
  );
  assert.doesNotMatch(page, /callout--warning/);
  assert.match(page, /role="tablist" aria-label="Campaign type"/);
  assert.match(page, /data-campaign-tab="registration"/);
  assert.match(page, /data-campaign-tab="referral"/);
  assert.match(page, /data-campaign-tab="member-drive"/);
  assert.match(page, /data-campaign-tab="jlr-contact"[^>]*>JLR Contact/);
  assert.match(page, /data-campaign-tab="september-survey"[^>]*>September Survey/);
  assert.match(page, /data-campaign-tab="freeform"/);
  assert.match(page, /role="tabpanel"[^>]+data-campaign-panel="registration"/);
  assert.match(page, /role="tabpanel"[^>]+data-campaign-panel="referral"/);
  assert.match(page, /role="tabpanel"[^>]+data-campaign-panel="jlr-contact"/);
  assert.match(page, /role="tabpanel"[^>]+data-campaign-panel="september-survey"/);
  assert.match(page, /role="tabpanel"[^>]+data-campaign-panel="freeform"/);
  assert.match(script, /\/api\/admin\/email-campaign-history/);
  assert.match(script, /\/api\/admin\/custom-campaign-preview/);
  assert.match(script, /\/api\/admin\/custom-campaign-send/);
  assert.match(script, /sourceCampaignId: sourceId/);
  assert.match(script, /Continue sending/);
  assert.match(script, /Edit draft/);
  assert.match(script, /campaign\.kind !== 'registration-reminder' && hasEditableCopy/);
  assert.match(script, /campaignTypeLabel\(campaign\.kind\)/);
  assert.match(script, /… ' \+ metrics\.length \+ ' more metrics/);
  assert.match(page, /more metrics/);
  assert.match(script, /event\.key === 'ArrowRight'/);
  assert.match(script, /event\.key === 'ArrowLeft'/);
  assert.match(script, /selectCampaignTab\('freeform'\)/);
  assert.match(page, /data-preview-endpoint="\/api\/admin\/jlr-contact-preview"/);
  assert.match(page, /data-preview-endpoint="\/api\/admin\/survey-campaign-preview"/);
  assert.match(specialisedCampaignBackend, /func AdminJLRContactPreview/);
  assert.match(specialisedCampaignBackend, /func AdminSurveyCampaignPreview/);
  assert.match(campaignBackend, /survey-september-2026/);
  assert.match(campaignBackend, /customCampaignRegisteredAudience/);
  assert.match(campaignBackend, /kind == surveyCampaignKind/);
  assert.match(specialisedCampaignBackend, /const emailCampaignBatchSize = 100/);
  assert.match(page, /All registered members/);
  assert.doesNotMatch(page, /Campaign library/);
  assert.match(script, /campaignTypeLabel\(campaign\.kind\) \+ ' · ' \+ campaign\.campaignId/);
  assert.doesNotMatch(script, /innerHTML\s*=/);
});

test('campaign history renders provider delivery feedback without recipient data', function () {
  assert.match(script, /Delivered/);
  assert.match(script, /Undeliverable/);
  assert.match(script, /Bounced/);
  assert.match(script, /Delayed/);
  assert.match(script, /Complaints/);
  assert.match(script, /feedbackRefreshedAt/);
});

test('portable homepage copy uses production links and live-value placeholders', function () {
  const copy = fs.readFileSync(path.join(root, 'docs/homepage-copy.md'), 'utf8');
  const joinUrl = ['https:', '', 'ipace-owners.org', 'join', ''].join('/');
  assert.ok(copy.includes(joinUrl));
  assert.match(copy, /\[Current owners joined\]/);
  assert.match(copy, /I-PACE owners working together for fair outcomes/);
  assert.doesNotMatch(copy, /\{[%{]/);
});
