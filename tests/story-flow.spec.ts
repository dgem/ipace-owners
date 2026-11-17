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
  await page.addScriptTag({ path: path.resolve('src/assets/js/story-renderer.js') });
  await page.addScriptTag({ path: path.resolve('src/assets/js/member-stories.js') });
  const editor = page.locator('[data-story-editor]');
  await editor.fill('First paragraph.\n\nSecond paragraph.');
  await page.locator('#story-media').setInputFiles(path.resolve('tests/fixtures/story-test.mp4'));
  await page.evaluate(() => { (document.activeElement as HTMLElement).blur(); window.getSelection()?.removeAllRanges(); });
  await page.getByRole('button', { name: 'Insert video' }).click();
  await expect(editor.locator('video')).toHaveCount(1);
  await expect(editor.locator('video')).toHaveAttribute('contenteditable', 'false');
  await editor.locator('video').evaluate((video: HTMLVideoElement) => video.play());
  await expect.poll(() => editor.locator('video').evaluate((video: HTMLVideoElement) => video.currentTime)).toBeGreaterThan(0);
  await editor.locator('video').evaluate((video: HTMLVideoElement) => video.pause());
  await page.getByRole('button', { name: 'Preview story' }).click();
  const preview = page.locator('[data-story-preview]');
  await expect(preview).toContainText('First paragraph.');
  const player = preview.locator('.story-markdown video');
  await expect(player).toHaveAttribute('controls', '');
  await player.evaluate((video: HTMLVideoElement) => video.play());
  await expect.poll(() => player.evaluate((video: HTMLVideoElement) => video.currentTime)).toBeGreaterThan(0);
  await page.getByRole('button', { name: 'Close preview' }).click();
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
  expect(saves[2].vehicleId).toBe('car');
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

test('public cards show a clean first paragraph rather than media Markdown', async ({ page }) => {
  await page.route('http://story.test/**', async route => {
    if (new URL(route.request().url()).pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-public-stories></main>' });
    return route.fulfill({ json: { stories: [{ id: 'example', title: 'An experience', displayName: 'Owner', bodyMarkdown: '## Introduction\n\n![Video](story-media:movie)\n\nFirst **paragraph** with _details_.\n\nSecond paragraph is not shown.', media: [] }] } });
  });
  await page.goto('http://story.test/');
  await page.addScriptTag({ path: path.resolve('src/assets/js/public-member-stories.js') });
  const card = page.locator('.story-grid .card');
  await expect(card).toContainText('First paragraph with details.');
  await expect(card).not.toContainText('story-media:');
  await expect(card).not.toContainText('Second paragraph');
  await expect(card.getByRole('link', { name: 'Read this story' })).toHaveAttribute('href', '/member-stories/example/');
});

test('admin must confirm before soft-deleting a story', async ({ page }) => {
  let deleted = false;
  await page.route('http://story.test/**', async route => {
    const req = route.request();
    const pathname = new URL(req.url()).pathname;
    if (pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-admin-stories></main>' });
    if (pathname === '/api/admin/stories' && req.method() === 'GET') return route.fulfill({ json: { stories: deleted ? [] : [{ id: 'story-1', title: 'My experience', displayName: 'Owner', status: 'draft', bodyMarkdown: 'A story', consent: {} }], total: deleted ? 0 : 1, page: 1, pageSize: 20 } });
    if (pathname === '/api/admin/stories' && req.method() === 'POST') {
      expect(req.postDataJSON()).toEqual({ id: 'story-1', action: 'delete', confirmation: 'DELETE' });
      deleted = true;
      return route.fulfill({ json: { ok: true } });
    }
    return route.abort();
  });
  await page.goto('http://story.test/');
  await page.evaluate(() => { (window as any).ipaceGetIdentityToken = () => Promise.resolve('token'); });
  await page.addScriptTag({ path: path.resolve('src/assets/js/story-renderer.js') });
  await page.addScriptTag({ path: path.resolve('src/assets/js/admin-member-stories.js') });
  await page.getByRole('button', { name: 'My experience' }).click();
  await expect(page.locator('[data-story-list-view]')).toBeHidden();
  await page.locator('[data-story-detail] [name=title]').fill('Unsaved admin edit');
  await page.locator('[name=bodyMarkdown]').fill('## Updated heading\n\nUnsaved **bold** text');
  await page.getByRole('button', { name: 'Preview story' }).click();
  await expect(page.locator('[data-story-preview] h2', { hasText: 'Updated heading' })).toBeVisible();
  await expect(page.locator('[data-story-preview] strong')).toHaveText('bold');
  await page.getByRole('button', { name: 'Close preview' }).click();
  page.once('dialog', dialog => dialog.dismiss());
  await page.getByRole('button', { name: 'Back to experiences' }).click();
  await expect(page.locator('[data-story-list-view]')).toBeHidden();
  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button', { name: 'Back to experiences' }).click();
  await expect(page.locator('[data-story-list-view]')).toBeVisible();
  await page.getByRole('button', { name: 'My experience' }).click();
  await page.getByRole('button', { name: 'Delete', exact: true }).click();
  await page.getByLabel('Type DELETE to confirm').fill('delete');
  await page.getByRole('button', { name: 'Confirm deletion' }).click();
  await expect(page.getByText('Type DELETE to confirm.')).toBeVisible();
  expect(deleted).toBe(false);
  await page.getByLabel('Type DELETE to confirm').fill('DELETE');
  await page.getByRole('button', { name: 'Confirm deletion' }).click();
  await expect(page.getByText('No experiences match these filters.')).toBeVisible();
  expect(deleted).toBe(true);
});

test('dismissed empty-survey notice stays hidden while a new survey appears', async ({ page }) => {
  let dismissed: string[] = [];
  let hasSurvey = false;
  await page.route('http://story.test/**', async route => {
    const req = route.request();
    const pathname = new URL(req.url()).pathname;
    if (pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-auth-container><section data-member-survey-summary></section></main>' });
    if (pathname === '/api/member/notifications') {
      if (req.method() === 'POST') dismissed.push(req.postDataJSON().id);
      return route.fulfill({ json: { dismissed } });
    }
    if (pathname === '/api/member/surveys') return route.fulfill({ json: { surveys: hasSurvey ? [{ canRespond: true, survey: { id: 'new-survey', title: 'New survey' } }] : [] } });
    return route.abort();
  });
  await page.goto('http://story.test/');
  await page.evaluate(() => {
    (window as any).ipaceGetIdentityToken = () => Promise.resolve('token');
    (window as any).firebase = { auth: () => ({ currentUser: { getIdToken: () => Promise.resolve('token') } }) };
  });
  await page.addScriptTag({ path: path.resolve('src/assets/js/member-notifications.js') });
  await page.addScriptTag({ path: path.resolve('src/assets/js/surveys.js') });
  await page.evaluate(() => document.dispatchEvent(new CustomEvent('member:data')));
  const notice = page.locator('[data-member-survey-summary]');
  await expect(notice).toContainText('There are no open surveys');
  await notice.getByRole('button', { name: 'Dismiss this notice' }).click();
  await expect(notice).toBeHidden();
  expect(dismissed).toEqual(['survey:none:v1']);
  hasSurvey = true;
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(notice).toBeVisible();
  await expect(notice).toContainText('New survey');
});

test('members with stories open a separate editor and return to their list', async ({ page }) => {
  await page.route('http://story.test/**', async route => {
    if (new URL(route.request().url()).pathname === '/') return route.fulfill({ contentType: 'text/html', body: '<main data-member-stories></main>' });
    return route.fulfill({ json: { eligible: true, displayName: 'Owner', vehicles: [{ id: 'car', vehicle: {} }], stories: [{ id: 'story-1', title: 'First story', status: 'draft', bodyMarkdown: 'Existing experience' }], limits: { images: 5, videos: 2, totalBytes: 524288000 } } });
  });
  await page.goto('http://story.test/');
  await page.evaluate(() => { (window as any).ipaceGetIdentityToken = () => Promise.resolve('token'); (window as any).ipaceIdentityUser = {}; });
  await page.addScriptTag({ path: path.resolve('src/assets/js/member-stories.js') });
  await expect(page.getByText('First story')).toBeVisible();
  await expect(page.locator('[data-story-form]')).toHaveCount(0);
  await page.getByRole('button', { name: 'Edit', exact: true }).click();
  await expect(page.locator('[data-story-editor]')).toContainText('Existing experience');
  await expect(page.getByText('First story', { exact: true })).toHaveCount(0);
  await page.getByRole('button', { name: 'Back to experiences' }).click();
  await page.getByRole('button', { name: 'Write a new experience' }).click();
  await expect(page.locator('[data-story-editor]')).toBeEmpty();
  await page.locator('[data-story-editor]').fill('Unsaved change');
  page.once('dialog', dialog => dialog.dismiss());
  await page.getByRole('button', { name: 'Back to experiences' }).click();
  await expect(page.locator('[data-story-editor]')).toContainText('Unsaved change');
  page.once('dialog', dialog => dialog.accept());
  await page.getByRole('button', { name: 'Back to experiences' }).click();
  await expect(page.getByText('First story')).toBeVisible();
});
