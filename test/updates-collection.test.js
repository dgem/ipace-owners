const test = require('node:test');
const assert = require('node:assert/strict');

test('built Updates listing includes Nunjucks and Markdown posts newest first', async () => {
  const { default: Eleventy } = await import('@11ty/eleventy');
  const site = new Eleventy('src', '_site', { quietMode: true });
  const pages = await site.toJSON();
  const listing = pages.find((page) => page.url === '/updates/');
  assert.ok(listing, 'Updates listing must be generated');
  const reminder = 'href="/updates/survey-final-straight/"';
  const olderPost = 'href="/updates/member-data-export/"';
  assert.ok(listing.content.includes(reminder), 'Nunjucks survey update must be discoverable');
  assert.ok(listing.content.includes(olderPost), 'Markdown updates must remain listed');
  assert.ok(listing.content.indexOf(reminder) < listing.content.indexOf(olderPost), 'newer update must appear first');
  assert.ok(pages.some((page) => page.url === '/updates/survey-final-straight/'), 'listed update must also have a page');
});
