import { expect, test, type Page } from '@playwright/test';

async function expectNoHorizontalOverflow(page: Page) {
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth))
    .toBe(true);
}

test('the homepage navigation is usable without horizontal overflow', async ({ page }, testInfo) => {
  await page.goto('/');

  await expect(page.getByRole('heading', { name: /owners working together for fair outcomes/i })).toBeVisible();
  await expectNoHorizontalOverflow(page);

  if (testInfo.project.name.includes('mobile')) {
    const menuButton = page.getByRole('button', { name: 'Open navigation menu' });
    await expect(menuButton).toBeVisible();
    await menuButton.click();

    await expect(page.getByRole('navigation', { name: 'Mobile navigation' })).toBeVisible();
    await expect(page.getByRole('link', { name: 'Join the Group' }).last()).toBeVisible();
    await expectNoHorizontalOverflow(page);
    return;
  }

  await expect(page.getByRole('navigation', { name: 'Main navigation' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Join the Group' }).first()).toBeVisible();
});

test('the member sign-in gate remains legible and usable', async ({ page }) => {
  await page.goto('/member/account/');

  await expect(page.getByRole('heading', { name: 'Sign in to view your account' })).toBeVisible();
  await expect(page.getByLabel('Email address')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Email me a sign-in link' })).toBeVisible();
  await expectNoHorizontalOverflow(page);
});
