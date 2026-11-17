import { test, expect } from '@playwright/test';
import path from 'node:path';

test('uploads a movie, retries a failure without duplicating the draft, and saves its position', async ({ page }) => {
  const saves: any[] = [];
  let uploads = 0;
  await page.route('http://story.test/**', async route => {
    const request = route.request();
    const url = new URL(request.url());
    if (url.pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-member-stories></main>' });
    if (url.pathname === '/api/member/stories' && request.method() === 'GET') return route.fulfill({ json: { eligible: true, displayName: 'Owner', vehicles: [{ id: 'car', vehicle: {} }], stories: [], limits: { images: 5, videos: 2, totalBytes: 524288000 } } });
    if (url.pathname === '/api/member/stories') { const data = request.postDataJSON(); saves.push(data); if (data.bodyMarkdown.includes('story-media:movie-1') && !data.mediaIds.includes('movie-1')) return route.fulfill({ status: 400, json: { error: 'Media links may only reference photos or videos uploaded to this experience' } }); return route.fulfill({ json: { story: { id: 'draft-1' } } }); }
    if (url.pathname === '/api/member/story-upload') return route.fulfill({ json: { mediaId: 'movie-1', uploadUrl: 'http://story.test/upload', contentType: 'video/quicktime' } });
    if (url.pathname === '/upload') { uploads++; return route.fulfill({ status: uploads === 1 ? 500 : 200, body: '' }); }
    if (url.pathname === '/api/member/story-upload-complete') return route.fulfill({ json: { ok: true } });
    return route.abort();
  });
  await page.goto('http://story.test/');
  await page.evaluate(() => { (window as any).ipaceGetIdentityToken = () => Promise.resolve('token'); (window as any).ipaceIdentityUser = {}; });
  await page.addScriptTag({ path: path.resolve('src/assets/js/member-stories.js') });
  const editor = page.locator('[data-story-editor]');
  await editor.fill('First paragraph.\n\nSecond paragraph.');
  await page.locator('#story-media').setInputFiles({ name: 'sample.mov', mimeType: 'video/quicktime', buffer: Buffer.from('sample movie') });
  await page.getByRole('button', { name: 'Insert video' }).click();
  await page.locator('[data-move-pending][data-direction="up"]').click();
  await expect(editor.locator('video')).toHaveCount(1);
  await page.getByRole('button', { name: 'Save draft' }).click();
  await expect(page.locator('[data-story-status]')).toContainText('Could not upload');
  await page.getByRole('button', { name: 'Save draft' }).click();
  await expect.poll(() => saves.length).toBe(3);
  expect(saves[0].id).toBe('');
  expect(saves[1].id).toBe('draft-1');
  expect(saves[2].id).toBe('draft-1');
  expect(saves[2].mediaIds).toEqual(['movie-1']);
  expect(saves[2].bodyMarkdown).toContain('![Video](story-media:movie-1)');
  expect(uploads).toBe(2);
});

test('member confirms deletion before removing an experience', async ({ page }) => {
  let deleted = false;
  await page.route('http://story.test/**', async route => {
    const req = route.request();
    const pathname = new URL(req.url()).pathname;
    if (pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-member-stories></main>' });
    if (pathname === '/api/member/stories' && req.method() === 'GET') return route.fulfill({ json: { eligible: true, displayName: 'Owner', vehicles: [{ id: 'car', vehicle: {} }], stories: deleted ? [] : [{ id: 'story-1', title: 'My experience', status: 'published' }], limits: { images: 5, videos: 2, totalBytes: 524288000 } } });
    if (pathname === '/api/member/stories' && req.method() === 'DELETE') {
      expect(req.postDataJSON()).toEqual({ id: 'story-1', confirmation: 'DELETE' });
      deleted = true;
      return route.fulfill({ json: { ok: true } });
    }
    return route.abort();
  });
  await page.goto('http://story.test/');
  await page.evaluate(() => { (window as any).ipaceGetIdentityToken = () => Promise.resolve('token'); (window as any).ipaceIdentityUser = {}; });
  await page.addScriptTag({ path: path.resolve('src/assets/js/member-stories.js') });
  await page.getByRole('button', { name: 'Delete', exact: true }).click();
  await page.getByLabel('Type DELETE to confirm').fill('delete');
  await page.getByRole('button', { name: 'Confirm deletion' }).click();
  await expect(page.getByText('Type DELETE to confirm.')).toBeVisible();
  expect(deleted).toBe(false);
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('button', { name: 'Confirm deletion' }).click();
  await expect(page.getByText('My experience', { exact: true })).toHaveCount(0);
  expect(deleted).toBe(true);
});
