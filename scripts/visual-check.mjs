/* global document, getComputedStyle, window */

import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { chromium } from 'playwright-core';

const baseURL = (process.env.VISUAL_BASE_URL || 'http://127.0.0.1:8080').replace(/\/$/, '');
const outputDir = process.env.VISUAL_OUTPUT_DIR || 'visual-artifacts';
const executablePath = [
  process.env.CHROME_PATH,
  '/usr/bin/google-chrome',
  '/usr/bin/google-chrome-stable',
  '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome',
  '/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary'
].filter(Boolean).find((candidate) => fs.existsSync(candidate));

assert.ok(executablePath, 'Chrome executable not found; set CHROME_PATH');
fs.mkdirSync(outputDir, { recursive: true });

let siteReady = false;
for (let attempt = 0; attempt < 30; attempt += 1) {
  try {
    const response = await fetch(baseURL + '/admin/outreach/');
    if (response.ok) { siteReady = true; break; }
  } catch {
    // The Eleventy server may still be starting.
  }
  await new Promise((resolve) => setTimeout(resolve, 1000));
}
assert.equal(siteReady, true, `site did not become ready at ${baseURL}`);
const browser = await chromium.launch({ executablePath, headless: true });
const expectedAdminDestinations = [
  '/admin/',
  '/admin/review-queue/',
  '/admin/outreach/',
  '/admin/email-campaigns/',
  '/admin/instagram-campaigns/'
];

async function assertAdminDestinations(locator) {
  assert.deepEqual(await locator.evaluateAll((links) => links.map((link) => link.getAttribute('href'))), expectedAdminDestinations);
}

async function revealAdminState(page) {
  await page.evaluate(function () {
    document.querySelectorAll('[data-requires-admin]').forEach(function (element) {
      element.style.display = '';
    });
    document.querySelectorAll('[data-admin-content]').forEach(function (element) {
      element.hidden = false;
    });
    document.querySelectorAll('[data-auth-pending], [data-auth-login-gate], [data-admin-only-gate]').forEach(function (element) {
      element.hidden = true;
    });
    document.querySelectorAll('.cookie-notice').forEach(function (element) {
      element.hidden = true;
    });
  });
}

async function checkDesktopAdminHeader() {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  await page.goto(baseURL + '/admin/outreach/', { waitUntil: 'networkidle' });
  await revealAdminState(page);

  const header = page.locator('.site-header');
  const primary = page.locator('.site-header__inner');
  const admin = page.locator('.site-admin-nav');
  const title = page.locator('.page-header');
  await admin.waitFor({ state: 'visible' });
  assert.equal(await admin.evaluate((element) => getComputedStyle(element).display), 'flex');
  await assertAdminDestinations(admin.locator('a'));

  const headerBox = await header.boundingBox();
  const primaryBox = await primary.boundingBox();
  const adminBox = await admin.boundingBox();
  const titleBox = await title.boundingBox();
  assert.ok(headerBox && primaryBox && adminBox && titleBox);
  assert.ok(adminBox.y >= primaryBox.y + primaryBox.height, 'admin row must sit below the primary header row');
  assert.ok(adminBox.y + adminBox.height <= headerBox.y + headerBox.height, 'header must expand around the admin row');
  assert.ok(titleBox.y >= headerBox.y + headerBox.height, 'page title must start below the expanded header');
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-desktop.png'), fullPage: true });
  await page.close();
}

async function checkMobileAdminDrawer() {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await page.goto(baseURL + '/admin/outreach/', { waitUntil: 'networkidle' });
  await revealAdminState(page);
  assert.equal(await page.locator('#identity-mobile-header-login-btn').isVisible(), true);
  await page.locator('#mobile-menu-toggle').click();
  await page.locator('.mobile-nav__admin').waitFor({ state: 'visible' });
  assert.equal(await page.locator('#identity-mobile-login-btn').isVisible(), true);
  assert.equal(await page.locator('.site-admin-nav').isVisible(), false);
  await assertAdminDestinations(page.locator('.mobile-nav__admin a'));
  const mobileLogin = page.locator('#identity-mobile-login-btn');
  assert.equal(await mobileLogin.isVisible(), true, 'signed-out mobile navigation must show Sign in');
  assert.equal(await mobileLogin.evaluate((element) => getComputedStyle(element).color), 'rgb(255, 255, 255)');
  assert.equal(await page.locator('.mobile-nav__identity-label').textContent(), 'Account');
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-mobile.png'), fullPage: true });
  await page.evaluate(() => {
    document.querySelector('#identity-mobile-header-login-btn').style.display = 'none';
    document.querySelector('#identity-mobile-login-btn').style.display = 'none';
    document.querySelector('#identity-mobile-logout-btn').style.display = '';
    document.querySelectorAll('.mobile-nav__identity [data-requires-auth]').forEach((element) => {
      element.style.display = '';
    });
  });
  assert.equal(await page.locator('.mobile-nav__identity a[href="/member/dashboard/"]').isVisible(), true);
  assert.equal(await page.locator('.mobile-nav__identity a[href="/member/account/"][data-requires-auth]').isVisible(), true);
  assert.equal(await page.locator('#identity-mobile-logout-btn').isVisible(), true);
  assert.equal(await page.locator('#identity-mobile-header-login-btn').isVisible(), false);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-mobile-signed-in.png'), fullPage: true });
  await page.close();
}

async function checkCampaignControls(viewport, screenshotName) {
  const page = await browser.newPage({ viewport });
  const placeholders = [
    'membersJoined', 'membersVerified', 'memberFirstName', 'memberLastName', 'memberTittle',
    'memberTitle', 'memberJoined', 'memberVerified', 'memberVehicles',
    'vehiclesRegisteredCount', 'vehiclesSoHReadingsCount'
  ];
  await page.route('**/api/admin/email-campaign-history', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      placeholders,
      feedbackAvailable: true,
      feedbackRefreshedAt: '2026-07-27T12:05:00Z',
      campaigns: [{
        campaignId: 'email-campaign_example',
        kind: 'custom-member',
        name: 'July member update',
        subject: 'A member update for {{memberFirstName}}',
        markdown: 'Hi {{memberFirstName}},\n\nWe now have {{membersJoined}} members.',
        status: 'sending',
        eligible: 389,
        sent: 371,
        delivered: 360,
        opened: 244,
        clicked: 91,
        awaitingDelivery: 3,
        bounced: 4,
        suppressed: 1,
        providerFailed: 0,
        delayed: 3,
        complained: 1,
        undeliverable: 5,
        failed: 1,
        remaining: 18,
        batchCount: 38,
        updatedAt: '2026-07-24T12:00:00Z'
      }]
    })
  }));
  await page.route('**/api/admin/custom-campaign-preview', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      campaignId: 'email-campaign_preview',
      sourceCampaignId: 'email-campaign_example',
      name: 'July member update — rerun',
      eligible: 389,
      sent: 0,
      failed: 0,
      batchSent: 0,
      remaining: 389,
      emailPreview: {
        subject: 'A member update for Alex',
        html: '<!doctype html><html lang="en"><body style="font-family:Arial;background:#f7f8fb;padding:24px"><main style="max-width:600px;margin:auto;background:white;padding:32px"><h1 style="color:#12324a">A member update for Alex</h1><p>Hi Alex,</p><p>We now have 412 members.</p></main></body></html>',
        text: 'Hi Alex,\n\nWe now have 412 members.'
      }
    })
  }));
  await page.route('**/api/admin/all-members-drive-preview', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      campaignId: 'all-members-drive-staging-2026-07-27',
      eligible: 412,
      registered: 0,
      sent: 0,
      batchSent: 0,
      remaining: 412,
      emailPreview: {
        subject: 'Thanks for joining — help us reach 1,000 I-PACE owners',
        html: '<!doctype html><html lang="en"><body style="font-family:Arial;background:#f7f8fb;padding:24px"><main style="max-width:600px;margin:auto;background:white;padding:32px"><h1 style="color:#12324a">Thank you for joining and adding your voice</h1><p>We launched on 17 July—less than two weeks ago.</p><p>I-PACE owners are stronger together.</p><h2>Suggested text to share</h2><p>Own an I-PACE? Add your voice and help us reach 1,000 members. Free to join: https://ipace-owners.org/</p><p><a href="https://www.facebook.com/">Facebook</a> <a href="https://wa.me/">WhatsApp</a></p></main></body></html>',
        text: 'Thank you for joining and adding your voice. We launched on 17 July—less than two weeks ago.\\n\\nI-PACE owners are stronger together.\\n\\nSuggested text to share\\n\\nOwn an I-PACE? Add your voice and help us reach 1,000 members. Free to join: https://ipace-owners.org/'
      }
    })
  }));
  await page.goto(baseURL + '/admin/email-campaigns/', { waitUntil: 'networkidle' });
  await revealAdminState(page);
  await page.evaluate(() => {
    window.firebase = { auth: () => ({ currentUser: { getIdToken: async () => 'visual-admin-token' } }) };
  });
  assert.equal(await page.locator('[data-email-campaign]').count(), 3);
  assert.equal(await page.locator('[data-custom-email-campaign]').count(), 1);
  assert.equal(await page.locator('[data-custom-placeholder]').count(), 11);
  assert.equal(await page.locator('[data-campaign-tab="registration"]').getAttribute('aria-selected'), 'true');
  assert.equal(await page.locator('[data-campaign-panel="registration"]').isVisible(), true);
  assert.equal(await page.locator('[data-campaign-panel="freeform"]').isVisible(), false);
  await page.locator('[data-campaign-open-tab="freeform"]').click();
  assert.equal(await page.locator('[data-campaign-tab="freeform"]').getAttribute('aria-selected'), 'true');
  assert.equal(await page.locator('[data-custom-campaign-markdown]').isVisible(), true);
  assert.equal(await page.locator('[data-custom-campaign-send-button]').isDisabled(), true);
  await page.locator('[data-campaign-history-refresh]').click();
  await page.locator('.email-campaign-history__item').waitFor({ state: 'visible' });
  assert.equal(await page.locator('.email-campaign-history__item').count(), 1);
  assert.equal(await page.locator('.email-campaign-history__item button').count(), 2);
  await page.locator('.email-campaign-history__item .btn--secondary').click();
  assert.equal(await page.locator('[data-custom-campaign-name]').inputValue(), 'July member update — rerun');
  await page.locator('[data-custom-campaign-preview]').click();
  await page.locator('[data-custom-campaign-email-preview]').waitFor({ state: 'visible' });
  await page.waitForFunction(() => !document.querySelector('[data-custom-campaign-send-button]').disabled);
  assert.equal(await page.locator('[data-custom-campaign-send-button]').isDisabled(), false);
  await page.locator('[data-custom-campaign-subject]').fill('A revised member update');
  assert.equal(await page.locator('[data-custom-campaign-send-button]').isDisabled(), true, 'editing must invalidate the preview');
  await page.locator('[data-custom-campaign-preview]').click();
  await page.waitForFunction(() => !document.querySelector('[data-custom-campaign-send-button]').disabled);
  await page.frameLocator('[data-custom-campaign-email-html]').locator('h1').waitFor({ state: 'visible' });
  assert.equal(await page.locator('[data-custom-campaign-send-button]').isDisabled(), false);
  const buttons = page.locator('[data-campaign-send-button]');
  assert.equal(await buttons.count(), 3);
  for (let index = 0; index < await buttons.count(); index += 1) {
    const button = buttons.nth(index);
    const panelName = await button.locator('xpath=ancestor::*[@data-campaign-panel]').getAttribute('data-campaign-panel');
    await page.locator(`[data-campaign-tab="${panelName}"]`).focus();
    await page.keyboard.press('Enter');
    await button.scrollIntoViewIfNeeded();
    assert.equal(await button.isVisible(), true, 'send controls must be visible before preview');
    assert.equal(await button.isDisabled(), true, 'send controls must remain disabled before preview');
    assert.ok(Number(await button.evaluate((element) => getComputedStyle(element).opacity)) < 0.8, 'disabled send controls must look disabled');
  }
  await page.locator('[data-campaign-tab="registration"]').focus();
  await page.keyboard.press('ArrowRight');
  assert.equal(await page.locator('[data-campaign-tab="referral"]').getAttribute('aria-selected'), 'true');
  assert.equal(await page.locator('[data-campaign-panel="referral"]').isVisible(), true);
  await page.keyboard.press('End');
  assert.equal(await page.locator('[data-campaign-tab="freeform"]').getAttribute('aria-selected'), 'true');
  await page.locator('[data-campaign-tab="registration"]').focus();
  await page.keyboard.press('Enter');
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(outputDir, screenshotName), fullPage: true });
  assert.equal(
    await page.locator('.email-campaign-history__item').evaluate((element) => element.getBoundingClientRect().right <= document.documentElement.clientWidth),
    true,
    'campaign history must fit within the viewport'
  );
  await page.locator('[data-campaign-tab="freeform"]').focus();
  await page.keyboard.press('Enter');
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(outputDir, screenshotName.replace('.png', '-freeform.png')), fullPage: true });
  await page.locator('[data-campaign-tab="member-drive"]').focus();
  await page.keyboard.press('Enter');
  await page.locator('[data-campaign-panel="member-drive"] [data-campaign-preview]').click();
  await page.frameLocator('[data-campaign-panel="member-drive"] [data-campaign-email-html]').getByText('Suggested text to share').waitFor({ state: 'visible' });
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(outputDir, screenshotName.replace('.png', '-member-drive.png')), fullPage: true });
  await page.close();
}

async function checkAdminDashboard() {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  await page.route('**/api/admin/campaign-summary', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      email: { available: true, runs: 4, sent: 412, delivered: 401, opened: 278, clicked: 103, awaitingDelivery: 3, bounced: 4, suppressed: 1, providerFailed: 0, delayed: 3, undeliverable: 5, failed: 1, remaining: 8 },
      instagram: { available: true, runs: 3, published: 2, drafts: 1, views: 3200, reach: 2400, totalInteractions: 186 },
      facebook: { available: false, message: 'Manual outreach only. Facebook Page Insights are not connected.' }
    })
  }));
  await page.goto(baseURL + '/admin/', { waitUntil: 'networkidle' });
  await page.evaluate(() => {
    window.firebase = { auth: () => ({ currentUser: { getIdToken: async () => 'visual-admin-token' } }) };
  });
  await revealAdminState(page);
  await page.locator('[data-campaign-summary-refresh]').click();
  await page.locator('.campaign-summary-card').first().waitFor({ state: 'visible' });
  assert.equal(await page.locator('.campaign-summary-card').count(), 3);
  assert.equal(await page.locator('.admin-dashboard-grid .card').count(), 4);
  assert.deepEqual(await page.locator('.admin-dashboard-grid a').evaluateAll((links) => links.map((link) => link.getAttribute('href'))), expectedAdminDestinations.slice(1));
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-dashboard-desktop.png'), fullPage: true });
  await page.close();
}

async function checkInstagramCampaigns(viewport, screenshotName) {
  const page = await browser.newPage({ viewport });
  await page.route('**/api/admin/instagram-campaign-history', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      insightsConfigured: true,
      campaigns: [
        {
          campaignId: 'instagram-published-example',
          name: 'Owners together launch',
          status: 'published',
          mediaPath: '/ipace-owners-instagram-launch-reel.mp4',
          caption: 'I-PACE owners are stronger when we speak together.',
          updatedAt: '2026-07-26T12:00:00Z',
          insights: { available: true, views: 3200, reach: 2400, totalInteractions: 186 }
        },
        {
          campaignId: 'instagram-draft-example',
          name: 'August owner update',
          status: 'draft',
          mediaPath: '/ipace-owners-instagram-launch-reel.mp4',
          caption: 'An editable Instagram draft.',
          updatedAt: '2026-07-27T08:00:00Z',
          insights: { available: false }
        }
      ]
    })
  }));
  await page.goto(baseURL + '/admin/instagram-campaigns/', { waitUntil: 'networkidle' });
  await page.evaluate(() => {
    window.firebase = { auth: () => ({ currentUser: { getIdToken: async () => 'visual-admin-token' } }) };
  });
  await revealAdminState(page);
  await page.locator('[data-instagram-history-refresh]').click();
  await page.locator('.email-campaign-history__item').first().waitFor({ state: 'visible' });
  assert.equal(await page.locator('.email-campaign-history__item').count(), 2);
  assert.equal(await page.getByRole('button', { name: 'Edit and repost' }).count(), 1);
  assert.equal(await page.getByRole('button', { name: 'Edit draft' }).count(), 1);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, screenshotName), fullPage: true });
  await page.close();
}

try {
  await checkDesktopAdminHeader();
  await checkMobileAdminDrawer();
  await checkAdminDashboard();
  await checkCampaignControls({ width: 1440, height: 1100 }, 'admin-email-campaigns-desktop.png');
  await checkCampaignControls({ width: 390, height: 844 }, 'admin-email-campaigns-mobile.png');
  await checkInstagramCampaigns({ width: 1440, height: 1100 }, 'admin-instagram-campaigns-desktop.png');
  await checkInstagramCampaigns({ width: 390, height: 844 }, 'admin-instagram-campaigns-mobile.png');
  console.log(`Visual checks passed; screenshots written to ${outputDir}`);
} finally {
  await browser.close();
}
