import { expect, test, type Page } from '@playwright/test';
import path from 'node:path';

const resendAPIKey = process.env.E2E_RESEND_API_KEY;
const receivingInbox = process.env.E2E_RESEND_INBOX;
const hasLiveAuthConfiguration = Boolean(process.env.E2E_BASE_URL && resendAPIKey && receivingInbox);

type ReceivedEmail = {
  created_at?: string;
  id?: string;
  subject?: string;
  to?: string[];
};

type ReceivedEmailList = {
  data?: ReceivedEmail[];
};

type ReceivedEmailContent = {
  html?: string;
  text?: string;
};

async function resendRequest(path: string) {
  const response = await fetch(`https://api.resend.com${path}`, {
    headers: {
      Authorization: `Bearer ${resendAPIKey}`,
    },
  });

  if (!response.ok) {
    throw new Error('Could not read the configured test inbox.');
  }

  return response.json();
}

function magicLinkFromMessage(message: ReceivedEmailContent, previewURL: string) {
  const content = `${message.html || ''}\n${message.text || ''}`.replaceAll('&amp;', '&');
  const candidates = content.match(/https:\/\/[^\s"'<>]+/gi) || [];
  const expectedAccountURL = new URL('/member/account/', previewURL).href;
  const expectedOrigin = new URL(previewURL).origin;

  for (const candidate of candidates) {
    try {
      const actionURL = new URL(candidate);
      const query = actionURL.searchParams;
      if (
        actionURL.origin === expectedOrigin &&
        actionURL.pathname === '/auth/action' &&
        !actionURL.username &&
        !actionURL.password &&
        !actionURL.hash &&
        query.getAll('mode').length === 1 &&
        query.get('mode') === 'signIn' &&
        query.getAll('oobCode').length === 1 &&
        Boolean(query.get('oobCode')) &&
        query.getAll('apiKey').length === 1 &&
        Boolean(query.get('apiKey')) &&
        query.getAll('continueUrl').length === 1 &&
        query.get('continueUrl') === expectedAccountURL
      ) {
        return candidate;
      }
    } catch {
      // Ignore malformed or unexpected links without exposing their contents.
    }
  }

  return undefined;
}

async function waitForMagicLink(notBefore: number) {
  for (let attempt = 0; attempt < 18; attempt += 1) {
    const messages = (await resendRequest('/emails/receiving')) as ReceivedEmailList;
    const matchingMessage = (messages.data || []).find((message) => {
      const receivedAt = Date.parse(message.created_at || '');
      const sentToInbox = (message.to || []).some((address) => address.toLowerCase() === receivingInbox?.toLowerCase());

      return Boolean(
        message.id &&
          sentToInbox &&
          /sign-in/i.test(message.subject || '') &&
          Number.isFinite(receivedAt) &&
          receivedAt >= notBefore - 5_000,
      );
    });

    if (matchingMessage?.id) {
      const message = (await resendRequest(`/emails/receiving/${matchingMessage.id}`)) as ReceivedEmailContent;
      const magicLink = magicLinkFromMessage(message, process.env.E2E_BASE_URL || '');

      if (magicLink) {
        return magicLink;
      }
    }

    await new Promise((resolve) => setTimeout(resolve, 5_000));
  }

  throw new Error('The secure sign-in email did not arrive in the configured test inbox.');
}

// This test account and its draft are staging-only. Never submit the draft for review.
async function verifyStoryUpload(page: Page) {
  const marker = 'Automated staging story upload test';
  await page.goto('/member/stories/');
  const root = page.locator('[data-member-stories]');
  await expect(root.locator('[data-story-form]'), 'The staging member needs story access, a name, display name and registered vehicle.').toBeVisible();
  const matches = root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) });
  expect(await matches.count(), 'The dedicated account must have at most one test draft.').toBeLessThanOrEqual(1);
  if (await matches.count()) {
    await matches.getByRole('button', { name: 'Edit' }).click();
    const removeButtons = root.locator('[data-remove-existing-media]');
    while (await removeButtons.count()) await removeButtons.first().click();
    await root.locator('[data-save-story]').click();
    await expect(root.locator('[data-story-form]')).toHaveAttribute('data-story-id', '');
    await expect(root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) })).toHaveCount(1);
    await root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) }).getByRole('button', { name: 'Edit' }).click();
  }
  const editor = root.locator('[data-story-editor]');
  await editor.fill(marker + '\n\nThis private staging draft exercises a real video upload and save.');
  await root.locator('#story-media').setInputFiles(path.resolve('tests/fixtures/story-test.mp4'));
  await root.getByRole('button', { name: 'Insert video' }).click();
  const saved = page.waitForResponse(response => response.url().endsWith('/api/member/stories') && response.request().method() === 'POST' && Boolean(response.request().postData()?.includes('story-media:')));
  await root.locator('[data-save-story]').click();
  expect((await saved).ok(), 'The final save must accept the uploaded media reference.').toBe(true);
  await expect(root.locator('[data-story-form]')).toHaveAttribute('data-story-id', '', { timeout: 60_000 });
  await expect(root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) })).toHaveCount(1, { timeout: 60_000 });
  const result = await page.evaluate(async (title) => {
    const token = await (window as any).ipaceGetIdentityToken();
    const response = await fetch('/api/member/stories', { headers: { Authorization: 'Bearer ' + token } });
    if (!response.ok) throw new Error('Unable to inspect the staging story.');
    const data = await response.json();
    const stories = data.stories.filter((story: any) => story.title === title);
    if (stories.length !== 1) throw new Error('Expected exactly one staging test draft.');
    const story = stories[0];
    const media = (story.media || []).filter((item: any) => item.status === 'ready' && item.contentType === 'video/mp4');
    if (story.status !== 'draft' || media.length !== 1 || !story.bodyMarkdown.includes('story-media:' + media[0].id)) throw new Error('The saved draft did not retain its uploaded video reference.');
    const video = await fetch(media[0].url);
    return { ok: video.ok, size: (await video.arrayBuffer()).byteLength };
  }, marker);
  expect(result.ok).toBe(true);
  expect(result.size).toBeGreaterThan(0);
}

test.describe('member magic-link sign-in', () => {
  if (process.env.CI && !hasLiveAuthConfiguration) {
    throw new Error('The enabled live authentication journey requires E2E_BASE_URL and dedicated Resend inbox credentials.');
  }
  test.skip(!hasLiveAuthConfiguration, 'Live authentication requires E2E_BASE_URL and the configured Resend test inbox.');

  test('a registered member can open a magic link and see protected account data', async ({ page }) => {
    test.setTimeout(240_000);
    await page.goto('/member/account/');
    await expect(page.getByRole('heading', { name: 'Sign in to view your account' })).toBeVisible();

    const requestedAt = Date.now();
    await page.getByLabel('Email address').fill(receivingInbox || '');
    await page.getByRole('button', { name: 'Email me a sign-in link' }).click();
    await expect(page.getByText(/secure sign-in link will be sent/i)).toBeVisible();

    const magicLink = await waitForMagicLink(requestedAt);
    try {
      await page.goto(magicLink);
    } catch {
      throw new Error('The secure sign-in link could not be opened.');
    }

    const expectedAccountURL = new URL('/member/account/', process.env.E2E_BASE_URL);
    await expect.poll(() => page.url()).toBe(expectedAccountURL.href);
    await expect(page.locator('[data-auth-content]')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Account details' })).toBeVisible();
    if (process.env.E2E_STORY_UPLOAD_ENABLED === 'true') {
      if (!process.env.E2E_STORY_ALLOWED_ORIGIN || new URL(process.env.E2E_BASE_URL || '').origin !== new URL(process.env.E2E_STORY_ALLOWED_ORIGIN).origin) throw new Error('Story upload requires an explicitly allowed staging origin.');
      await verifyStoryUpload(page);
    }
  });
});
