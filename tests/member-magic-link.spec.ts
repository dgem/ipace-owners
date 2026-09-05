import { expect, test } from '@playwright/test';

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

function magicLinkFromMessage(message: ReceivedEmailContent) {
  const content = `${message.html || ''}\n${message.text || ''}`.replaceAll('&amp;', '&');
  const candidates = content.match(/https:\/\/[^\s"'<>]+/gi) || [];

  return candidates.find((candidate) => /[?&](?:oobCode|mode=signIn)=/i.test(candidate));
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
      const magicLink = magicLinkFromMessage(message);

      if (magicLink) {
        return magicLink;
      }
    }

    await new Promise((resolve) => setTimeout(resolve, 5_000));
  }

  throw new Error('The secure sign-in email did not arrive in the configured test inbox.');
}

test.describe('member magic-link sign-in', () => {
  test.skip(!hasLiveAuthConfiguration, 'Live authentication requires E2E_BASE_URL and the configured Resend test inbox.');

  test('a registered member can open a magic link and see protected account data', async ({ page }) => {
    await page.goto('/member/account/');
    await expect(page.getByRole('heading', { name: 'Sign in to view your account' })).toBeVisible();

    const requestedAt = Date.now();
    await page.getByLabel('Email address').fill(receivingInbox || '');
    await page.getByRole('button', { name: 'Email me a sign-in link' }).click();
    await expect(page.getByText(/secure sign-in link will be sent/i)).toBeVisible();

    const magicLink = await waitForMagicLink(requestedAt);
    await page.goto(magicLink);

    await expect(page).toHaveURL(/\/member\/account\//);
    await expect(page.locator('[data-auth-content]')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Account details' })).toBeVisible();
  });
});
