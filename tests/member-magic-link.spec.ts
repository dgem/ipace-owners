import { expect, test, type Page } from '@playwright/test';
import path from 'node:path';
import fs from 'node:fs';
import { magicLinkFromMessage } from '../scripts/auth-e2e-link.cjs';
import { insertStoryExampleMedia, prepareStoryExample } from './story-example-helper';

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
  const example = fs.readFileSync(path.resolve('tests/fixtures/story-example.md'), 'utf8');
  const { body, placements } = prepareStoryExample(example);
  // Provision at most one synthetic vehicle on the dedicated test member.
  await page.evaluate(async () => {
    const token = await (window as any).ipaceGetIdentityToken();
    const headers = { Authorization: 'Bearer ' + token };
    const current = await fetch('/api/member/stories', { headers });
    if (!current.ok) throw new Error('The staging test member cannot access experiences.');
    const data = await current.json();
    if ((data.vehicles || []).length === 0) {
      const created = await fetch('/api/submit-vehicle-basics', {
        method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' },
        body: JSON.stringify({ registration: 'E2E TEST', country: 'GB', modelYear: '2020' }),
      });
      if (!created.ok) throw new Error('Could not create the synthetic staging test vehicle.');
    }
  });
  await page.goto('/member/stories/');
  // Keep Playwright's target stable when moving between distant media controls.
  await page.addStyleTag({ content: 'html { scroll-behavior: auto !important; }' });
  const root = page.locator('[data-member-stories]');
  await expect(root.locator('[data-story-form], [data-new-story]'), 'The staging member needs story access, a name, display name and registered vehicle.').toBeVisible();
  const matches = root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) });
  expect(await matches.count(), 'The dedicated account must have at most one test draft.').toBeLessThanOrEqual(1);
  if (await matches.count()) {
    await matches.getByRole('button', { name: 'Edit' }).click();
    const removeButtons = root.locator('[data-remove-existing-media]');
    while (await removeButtons.count()) await removeButtons.first().click();
    await root.locator('[data-save-story]').click();
    await expect(root.locator('[data-story-form]')).toHaveCount(0);
    await expect(root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) })).toHaveCount(1);
    await root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) }).getByRole('button', { name: 'Edit' }).click();
  } else if (await root.locator('[data-new-story]').count()) {
    await root.locator('[data-new-story]').click();
  }
  const editor = root.locator('[data-story-editor]');
  await editor.fill(marker + '\n\n' + body);
  await root.getByLabel('Publish this experience and selected media on this website.').check();
  await root.getByLabel('I own, or have permission to supply, uploaded media.').check();
  await root.getByLabel('This is true to the best of my knowledge.').check();
  await root.locator('#story-media').setInputFiles([
    path.resolve('public/images/racing-laurel-email.png'),
    path.resolve('tests/fixtures/story-test.mp4'),
  ]);
  await insertStoryExampleMedia(root, placements);
  await root.getByRole('button', { name: 'Preview story' }).click();
  await expect(root.locator('[data-story-preview] .story-markdown video')).toHaveCount(1);
  await expect(root.locator('[data-story-preview] .story-markdown img')).toHaveCount(2);
  await root.getByRole('button', { name: 'Close preview' }).click();
  const saved = page.waitForResponse(response => response.url().endsWith('/api/member/stories') && response.request().method() === 'POST' && Boolean(response.request().postData()?.includes('story-media:')), { timeout: 60_000 });
  await root.locator('[data-save-story]').click();
  const finalResponse = await saved;
  if (!finalResponse.ok()) { const body = await finalResponse.json().catch(() => ({})); throw new Error('Final story save failed (' + finalResponse.status() + '): ' + (body.error || 'unknown error')); }
  await expect(root.locator('[data-story-form]')).toHaveCount(0, { timeout: 60_000 });
  await expect(root.locator('.story-submission-list li').filter({ has: page.getByText(marker, { exact: true }) })).toHaveCount(1, { timeout: 60_000 });
  const result = await page.evaluate(async ({ title, expectedText }) => {
    const token = await (window as any).ipaceGetIdentityToken();
    const response = await fetch('/api/member/stories', { headers: { Authorization: 'Bearer ' + token } });
    if (!response.ok) throw new Error('Unable to inspect the staging story.');
    const data = await response.json();
    const stories = data.stories.filter((story: any) => story.title === title);
    if (stories.length !== 1) throw new Error('Expected exactly one staging test draft.');
    const story = stories[0];
    const media = (story.media || []).filter((item: any) => item.status === 'ready' && item.contentType === 'video/mp4');
    if (story.status !== 'draft' || !story.consent?.website || !story.consent?.copyright || !story.consent?.truthful || media.length !== 1 || !story.bodyMarkdown.includes('story-media:' + media[0].id)) throw new Error('The saved draft did not retain its publication permissions and uploaded video reference.');
    const images = (story.media || []).filter((item: any) => item.status === 'ready' && item.contentType.startsWith('image/'));
    const refs = Array.from(story.bodyMarkdown.matchAll(/!\[[^\]]*\]\(story-media:([a-zA-Z0-9_-]+)\)/g), (match: RegExpMatchArray) => match[1]);
    if (images.length !== 1 || refs.length !== 3 || refs[0] !== images[0].id || refs[1] !== media[0].id || refs[2] !== images[0].id) throw new Error('The draft did not retain the example media in the expected positions.');
    const actualText = story.bodyMarkdown.replace(/!\[[^\]]*\]\(story-media:[^)]+\)/g, '').replace(/\s+/g, ' ');
    for (const paragraph of expectedText) if (paragraph && !actualText.includes(paragraph)) throw new Error('The draft did not retain the example story text.');
    const video = await fetch(media[0].url);
    return { ok: video.ok, size: (await video.arrayBuffer()).byteLength };
  }, { title: marker, expectedText: body.split(/E2EMEDIAPOSITION\d+END/).map((part: string) => part.replace(/\s+/g, ' ').trim()) });
  expect(result.ok).toBe(true);
  expect(result.size).toBeGreaterThan(0);
}

test.describe('member magic-link sign-in', () => {
  // The workflow has a ten-minute job limit; retries can otherwise hide the
  // underlying timeout and terminate the job before Playwright reports it.
  test.describe.configure({ retries: 0 });
  if (process.env.CI && !hasLiveAuthConfiguration) {
    throw new Error('The enabled live authentication journey requires E2E_BASE_URL and dedicated Resend inbox credentials.');
  }
  test.skip(!hasLiveAuthConfiguration, 'Live authentication requires E2E_BASE_URL and the configured Resend test inbox.');

  test('a registered member can open a magic link and see protected account data', async ({ page }) => {
    test.setTimeout(240_000);
    page.setDefaultTimeout(15_000);
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
