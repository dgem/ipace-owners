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

async function revealAdminState(page) {
  await page.evaluate(function () {
    document.querySelectorAll('[data-requires-auth]').forEach(function (element) {
      element.style.display = '';
    });
    document.querySelectorAll('[data-requires-admin]').forEach(function (element) {
      element.style.display = '';
    });
    document.querySelectorAll('[data-requires-guest]').forEach(function (element) {
      element.style.display = 'none';
    });
    document.querySelectorAll('#identity-login-btn, #identity-mobile-header-login-btn, #identity-mobile-login-btn').forEach(function (element) {
      element.style.display = 'none';
    });
    document.querySelectorAll('#identity-logout-btn, #identity-mobile-logout-btn').forEach(function (element) {
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

async function assertAdminBreadcrumb(page, currentLabel) {
  const breadcrumb = page.locator('.site-context-nav');
  assert.equal(await breadcrumb.isVisible(), true);
  assert.equal(await breadcrumb.getAttribute('aria-label'), 'Admin breadcrumb');
  assert.equal(await breadcrumb.locator('.site-context-nav__current').textContent(), currentLabel);
  assert.equal(await breadcrumb.locator('.site-context-nav__current').getAttribute('aria-current'), 'page');
  assert.equal(await breadcrumb.locator('.site-context-nav__list').evaluate((element) => {
    const items = Array.from(element.children).map((item) => item.getBoundingClientRect());
    const firstCentre = items[0].top + (items[0].height / 2);
    return items.every((item) => Math.abs((item.top + (item.height / 2)) - firstCentre) < 1);
  }), true, 'admin breadcrumbs must remain on one line');
}

async function checkDesktopAdminHeader() {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  await page.goto(baseURL + '/admin/outreach/', { waitUntil: 'networkidle' });
  await revealAdminState(page);
  await assertAdminBreadcrumb(page, 'Facebook outreach');

  const header = page.locator('.site-header');
  const primary = page.locator('.site-header__inner');
  const title = page.locator('.page-header');
  const admin = page.locator('.site-header__actions a[href="/admin/"][data-requires-admin]');
  await admin.waitFor({ state: 'visible' });
  assert.equal(await admin.textContent(), 'Admin');
  assert.equal(await page.locator('.site-admin-nav').count(), 0);
  assert.equal(await page.locator('#identity-user-display').count(), 0);
  assert.equal(await page.locator('.site-header__actions a[href="/member/account/"][data-requires-auth]').textContent(), 'My Data');

  const headerBox = await header.boundingBox();
  const primaryBox = await primary.boundingBox();
  const titleBox = await title.boundingBox();
  assert.ok(headerBox && primaryBox && titleBox);
  assert.ok(headerBox.height <= 125, 'admin breadcrumb must keep the shared header compact');
  assert.ok(titleBox.y >= headerBox.y + headerBox.height, 'page title must start below the header');
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-desktop.png'), fullPage: true });
  await page.close();
}

async function checkMobileAdminDrawer() {
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  await page.goto(baseURL + '/admin/outreach/', { waitUntil: 'networkidle' });
  await assertAdminBreadcrumb(page, 'Facebook outreach');
  assert.equal(await page.locator('#identity-mobile-header-login-btn').isVisible(), true);
  await page.locator('#mobile-menu-toggle').click();
  assert.equal(await page.locator('#identity-mobile-login-btn').isVisible(), true);
  assert.equal(await page.locator('.mobile-nav__admin').count(), 0);
  assert.equal(await page.locator('.mobile-nav__identity a[href="/admin/"]').isVisible(), false);
  const mobileLogin = page.locator('#identity-mobile-login-btn');
  assert.equal(await mobileLogin.isVisible(), true, 'signed-out mobile navigation must show Sign in');
  assert.equal(await mobileLogin.evaluate((element) => getComputedStyle(element).color), 'rgb(255, 255, 255)');
  assert.equal(await page.locator('.mobile-nav__identity-label').textContent(), 'Member');
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-mobile.png'), fullPage: true });
  await revealAdminState(page);
  assert.equal(await page.locator('.mobile-nav__identity a[href="/member/account/"][data-requires-auth]').textContent(), 'My Data');
  assert.equal(await page.locator('.mobile-nav__identity a[href="/member/dashboard/"]').count(), 0);
  assert.equal(await page.locator('.mobile-nav__identity a[href="/admin/"]').isVisible(), true);
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
  await assertAdminBreadcrumb(page, 'Email campaigns');
  await page.evaluate(() => {
    window.firebase = { auth: () => ({ currentUser: { getIdToken: async () => 'visual-admin-token' } }) };
  });
  assert.equal(await page.locator('[data-email-campaign]').count(), 3);
  assert.equal(await page.locator('[data-custom-email-campaign]').count(), 1);
  assert.equal(await page.locator('[data-custom-placeholder]').count(), 11);
  if (viewport.width < 640) {
    assert.equal(await page.locator('[data-campaign-tab]').evaluateAll((tabs) => tabs.every((tab) => {
      const rect = tab.getBoundingClientRect();
      return rect.left >= 0 && rect.right <= document.documentElement.clientWidth;
    })), true, 'all campaign tabs must be fully visible on mobile');
  }
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
  await assertAdminBreadcrumb(page, 'Dashboard');
  await page.locator('[data-campaign-summary-refresh]').click();
  await page.locator('.campaign-summary-card').first().waitFor({ state: 'visible' });
  assert.equal(await page.locator('.campaign-summary-card').count(), 3);
  assert.equal(await page.locator('.admin-dashboard-grid .card').count(), 4);
  assert.equal(await page.locator('.admin-dashboard-grid .admin-tool-logo svg').count(), 4);
  assert.equal(await page.locator('.admin-dashboard-grid .btn--primary').count(), 4);
  assert.deepEqual(await page.locator('.admin-dashboard-grid .btn').allTextContents(), [
    'Review Queue', 'Facebook Assistant', 'Email Campaigns', 'Instagram Campaigns'
  ]);
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
  await assertAdminBreadcrumb(page, 'Instagram campaigns');
  await page.locator('[data-instagram-history-refresh]').click();
  await page.locator('.email-campaign-history__item').first().waitFor({ state: 'visible' });
  assert.equal(await page.locator('.email-campaign-history__item').count(), 2);
  assert.equal(await page.getByRole('button', { name: 'Edit and repost' }).count(), 1);
  assert.equal(await page.getByRole('button', { name: 'Edit draft' }).count(), 1);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, screenshotName), fullPage: true });
  await page.close();
}

async function checkMemberExport(viewport, screenshotName) {
  const page = await browser.newPage({ viewport });
  await page.goto(baseURL + '/member/account/', { waitUntil: 'networkidle' });
  const isMobile = viewport.width < 640;
  assert.equal(await page.locator('.site-context-nav').isVisible(), true);
  assert.equal(await page.locator('.site-context-nav__current').textContent(), 'Account');
  assert.equal(await page.locator('.site-context-nav__current').getAttribute('aria-current'), 'page');
  assert.equal(await page.locator('.site-context-nav__list').evaluate((element) => {
    const items = Array.from(element.children).map((item) => item.getBoundingClientRect());
    const firstCentre = items[0].top + (items[0].height / 2);
    return items.every((item) => Math.abs((item.top + (item.height / 2)) - firstCentre) < 1);
  }), true, 'member breadcrumbs must remain on one line');
  assert.equal(await page.locator('#identity-login-btn').isVisible(), !isMobile);
  assert.equal(await page.locator('#identity-mobile-header-login-btn').isVisible(), isMobile);
  await page.evaluate(() => {
    document.querySelectorAll('[data-auth-content]').forEach((element) => { element.hidden = false; });
    document.querySelectorAll('[data-auth-pending], [data-auth-login-gate]').forEach((element) => { element.hidden = true; });
    document.querySelectorAll('[data-requires-auth]').forEach((element) => { element.style.display = ''; });
    document.querySelectorAll('[data-requires-guest], #identity-login-btn, #identity-mobile-header-login-btn, #identity-mobile-login-btn').forEach((element) => { element.style.display = 'none'; });
    document.querySelectorAll('#identity-logout-btn, #identity-mobile-logout-btn').forEach((element) => { element.style.display = ''; });
    document.querySelectorAll('.cookie-notice').forEach((element) => { element.hidden = true; });
  });
  assert.equal(await page.locator('[data-member-export]').count(), 2);
  assert.equal(await page.getByRole('heading', { name: 'Export your data' }).isVisible(), true);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.locator('[data-member-export]').first().scrollIntoViewIfNeeded();
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(outputDir, screenshotName), fullPage: true });
  await page.close();
}

async function checkPublicContentPage(url, heading, viewport, screenshotName) {
  const page = await browser.newPage({ viewport });
  await page.goto(baseURL + url, { waitUntil: 'networkidle' });
  await page.evaluate(() => {
    document.querySelectorAll('.cookie-notice').forEach((element) => { element.hidden = true; });
  });
  assert.equal(await page.getByRole('heading', { level: 1, name: heading }).isVisible(), true);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, screenshotName), fullPage: true });
  await page.close();
}

async function checkPublicEvidenceCounters(url, viewport, screenshotName, showMemberCta = false) {
  const page = await browser.newPage({ viewport });
  await page.route('**/api/public-stats?v=6', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      schemaVersion: 6,
      generatedAt: '2026-07-29T09:30:00Z',
      joinedOwners: 429,
      registeredMembers: 401,
      ownersContributed: 287,
      vehiclesRegistered: 312,
      vehiclesWithSoh: 196,
      sohReadings: 634,
      serviceEventsLogged: 148,
      vehiclesWithRepeatSoh: 83,
      averageReportedSoh: 87.4,
      averageSohChange: -2.7,
      sohDistribution: [
        { label: '90-100%', count: 61 },
        { label: '80-89.9%', count: 102 },
        { label: '70-79.9%', count: 28 },
        { label: 'Below 70%', count: 5 }
      ],
      modelYearDistribution: [
        { label: '2019', count: 92 },
        { label: '2020', count: 118 },
        { label: '2021', count: 102 }
      ]
    })
  }));
  await page.goto(baseURL + url, { waitUntil: 'networkidle' });
  await page.evaluate(() => {
    document.querySelectorAll('.cookie-notice').forEach((element) => { element.hidden = true; });
  });
  if (showMemberCta) {
    await page.locator('.launch-hero__evidence-cta').evaluate((element) => {
      element.style.display = '';
    });
    assert.equal(await page.getByRole('link', { name: 'Add Your Vehicle Data' }).isVisible(), true);
    assert.equal(
      await page.getByRole('link', { name: 'Add Your Vehicle Data' }).getAttribute('href'),
      '/member/dashboard/'
    );
    if (viewport.width < 640) {
      assert.equal(await page.getByRole('link', { name: 'Add Your Vehicle Data' }).evaluate((element) => {
        const bounds = element.getBoundingClientRect();
        const heroBounds = element.closest('.launch-hero').getBoundingClientRect();
        return bounds.left >= 0
          && bounds.right <= document.documentElement.clientWidth
          && bounds.top >= heroBounds.top
          && bounds.bottom <= heroBounds.bottom;
      }), true, 'the signed-in evidence CTA must remain fully contained by the mobile hero');
    }
  }
  await page.locator('[data-public-stat="serviceEventsLogged"]').first().waitFor({ state: 'visible' });
  assert.equal(await page.locator('[data-public-stat="vehiclesRegistered"]').first().textContent(), '312');
  assert.equal(await page.locator('[data-public-stat="sohReadings"]').first().textContent(), '634');
  assert.equal(await page.locator('[data-public-stat="serviceEventsLogged"]').first().textContent(), '148');
  if (url === '/') {
    assert.equal(await page.locator('.launch-hero .hero__media > .launch-hero__evidence').count(), 1);
    assert.equal(await page.locator('.launch-evidence-wreaths .launch-member-count').count(), 3);
    assert.equal(await page.locator('.launch-evidence-wreaths .launch-member-count__laurels').count(), 3);
    const evidenceComposition = await page.evaluate(() => {
      const hero = document.querySelector('.launch-hero');
      const heroInner = hero.querySelector('.hero__inner');
      const media = heroInner.querySelector('.hero__media');
      const image = media.querySelector('img');
      const evidence = hero.querySelector('.launch-hero__evidence');
      const wreaths = Array.from(evidence.querySelectorAll('.launch-member-count'));
      const heroBounds = hero.getBoundingClientRect();
      const imageBounds = image.getBoundingClientRect();
      const evidenceBounds = evidence.getBoundingClientRect();
      const wreathBounds = wreaths.map((wreath) => wreath.getBoundingClientRect());
      const headlineWreathBounds = heroInner.querySelector('.launch-member-count').getBoundingClientRect();
      const memberCta = hero.querySelector('.launch-hero__evidence-cta');
      return {
        background: getComputedStyle(evidence).backgroundColor,
        borderTopWidth: getComputedStyle(evidence).borderTopWidth,
        ctaBorderTopWidth: getComputedStyle(memberCta).borderTopWidth,
        followsHeroImage: evidenceBounds.top >= imageBounds.bottom,
        belongsToMediaColumn: evidence.parentElement === media,
        containedByHero: evidenceBounds.bottom <= heroBounds.bottom,
        subordinateSize: wreathBounds.every((bounds) => bounds.width < headlineWreathBounds.width),
        oneRow: Math.max(...wreathBounds.map((bounds) => bounds.top))
          - Math.min(...wreathBounds.map((bounds) => bounds.top)) < 2
      };
    });
    assert.equal(evidenceComposition.background, 'rgba(0, 0, 0, 0)');
    assert.equal(evidenceComposition.borderTopWidth, '0px');
    assert.equal(evidenceComposition.ctaBorderTopWidth, '0px');
    assert.equal(evidenceComposition.followsHeroImage, true);
    assert.equal(evidenceComposition.belongsToMediaColumn, true);
    assert.equal(evidenceComposition.containedByHero, true);
    assert.equal(evidenceComposition.subordinateSize, true);
    assert.equal(evidenceComposition.oneRow, true);
  }
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
  await checkMemberExport({ width: 1440, height: 1000 }, 'member-export-desktop.png');
  await checkMemberExport({ width: 390, height: 844 }, 'member-export-mobile.png');
  await checkPublicContentPage('/updates/member-data-export/', 'Export your member and vehicle data', { width: 1440, height: 1000 }, 'member-export-update-desktop.png');
  await checkPublicContentPage('/updates/member-data-export/', 'Export your member and vehicle data', { width: 390, height: 844 }, 'member-export-update-mobile.png');
  await checkPublicContentPage('/privacy/', 'Privacy Policy', { width: 1440, height: 1000 }, 'privacy-desktop.png');
  await checkPublicContentPage('/privacy/', 'Privacy Policy', { width: 390, height: 844 }, 'privacy-mobile.png');
  await checkPublicEvidenceCounters('/', { width: 1440, height: 1000 }, 'public-evidence-counters-desktop.png');
  await checkPublicEvidenceCounters('/', { width: 390, height: 844 }, 'public-evidence-counters-mobile.png');
  await checkPublicEvidenceCounters('/', { width: 1440, height: 1000 }, 'public-evidence-member-cta-desktop.png', true);
  await checkPublicEvidenceCounters('/', { width: 390, height: 844 }, 'public-evidence-member-cta-mobile.png', true);
  await checkPublicEvidenceCounters('/', { width: 412, height: 915 }, 'public-evidence-member-cta-android-15.png', true);
  await checkPublicEvidenceCounters('/evidence-dashboard/?site-mode=full', { width: 1440, height: 1000 }, 'evidence-dashboard-desktop.png');
  await checkPublicEvidenceCounters('/evidence-dashboard/?site-mode=full', { width: 390, height: 844 }, 'evidence-dashboard-mobile.png');
  console.log(`Visual checks passed; screenshots written to ${outputDir}`);
} finally {
  await browser.close();
}
