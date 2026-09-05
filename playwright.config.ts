import { defineConfig, devices } from '@playwright/test';

const baseURL = (process.env.E2E_BASE_URL || 'http://127.0.0.1:4173').replace(/\/$/, '');
const runsOnlyLiveAuth = process.argv.includes('--project=member-magic-link');

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  reporter: process.env.CI ? [['github'], ['html', { open: 'never' }]] : 'list',
  timeout: 30_000,
  expect: {
    timeout: 10_000,
  },
  outputDir: 'test-results',
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off',
  },
  webServer: process.env.E2E_BASE_URL || runsOnlyLiveAuth
    ? undefined
    : {
        command: 'npm run dev -- --port=4173',
        url: baseURL,
        reuseExistingServer: !process.env.CI,
      },
  projects: [
    {
      name: 'responsive-chromium',
      testMatch: /public-responsive\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'responsive-firefox',
      testMatch: /public-responsive\.spec\.ts/,
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'responsive-webkit',
      testMatch: /public-responsive\.spec\.ts/,
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'responsive-mobile-chrome',
      testMatch: /public-responsive\.spec\.ts/,
      use: { ...devices['Pixel 7'] },
    },
    {
      name: 'responsive-mobile-safari',
      testMatch: /public-responsive\.spec\.ts/,
      use: { ...devices['iPhone 13'] },
    },
    {
      // This project consumes a real, short-lived magic link. Never retain
      // screenshots, videos, or traces that could contain the link or account data.
      name: 'member-magic-link',
      testMatch: /member-magic-link\.spec\.ts/,
      workers: 1,
      use: {
        ...devices['Desktop Chrome'],
        trace: 'off',
        screenshot: 'off',
        video: 'off',
      },
    },
  ],
});
