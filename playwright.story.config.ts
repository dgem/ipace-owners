import { defineConfig } from '@playwright/test';
export default defineConfig({
  testDir: './tests',
  testMatch: /story-flow\.spec\.ts/,
  use: { browserName: 'chromium', launchOptions: process.env.E2E_CHROMIUM_PATH ? { executablePath: process.env.E2E_CHROMIUM_PATH } : {} },
});
