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
  assert.equal(await admin.locator('a').count(), 4);

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
  await page.locator('#mobile-menu-toggle').click();
  await page.locator('.mobile-nav__admin').waitFor({ state: 'visible' });
  assert.equal(await page.locator('.site-admin-nav').isVisible(), false);
  assert.equal(await page.locator('.mobile-nav__admin a').count(), 4);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-outreach-mobile.png'), fullPage: true });
  await page.close();
}

async function checkCampaignControls() {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
  await page.goto(baseURL + '/admin/email-campaigns/', { waitUntil: 'networkidle' });
  await revealAdminState(page);
  assert.equal(await page.locator('[data-email-campaign]').count(), 2);
  const buttons = page.locator('[data-campaign-send-button]');
  assert.equal(await buttons.count(), 2);
  for (let index = 0; index < await buttons.count(); index += 1) {
    const button = buttons.nth(index);
    await button.scrollIntoViewIfNeeded();
    assert.equal(await button.isVisible(), true, 'send controls must be visible before preview');
    assert.equal(await button.isDisabled(), true, 'send controls must remain disabled before preview');
    assert.ok(Number(await button.evaluate((element) => getComputedStyle(element).opacity)) < 0.8, 'disabled send controls must look disabled');
  }
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({ path: path.join(outputDir, 'admin-email-campaigns-desktop.png'), fullPage: true });
  await page.close();
}

async function checkAdminDashboard() {
  const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
  await page.goto(baseURL + '/admin/', { waitUntil: 'networkidle' });
  await revealAdminState(page);
  assert.equal(await page.locator('.admin-dashboard-grid .card').count(), 3);
  assert.equal(await page.locator('.admin-dashboard-grid a').count(), 3);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth), true);
  await page.screenshot({ path: path.join(outputDir, 'admin-dashboard-desktop.png'), fullPage: true });
  await page.close();
}

try {
  await checkDesktopAdminHeader();
  await checkMobileAdminDrawer();
  await checkAdminDashboard();
  await checkCampaignControls();
  console.log(`Visual checks passed; screenshots written to ${outputDir}`);
} finally {
  await browser.close();
}
