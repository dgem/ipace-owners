const test = require('node:test');
const assert = require('node:assert/strict');

test('built Updates listing includes Nunjucks and Markdown posts newest first', async () => {
  const { default: Eleventy } = await import('@11ty/eleventy');
  const site = new Eleventy('src', '_site', { quietMode: true });
  const pages = await site.toJSON();
  const listing = pages.find((page) => page.url === '/updates/');
  assert.ok(listing, 'Updates listing must be generated');
  const meeting = 'href="/updates/after-our-first-jlr-meeting/"';
  const results = 'href="/updates/september-survey-results/"';
  const reminder = 'href="/updates/survey-final-straight/"';
  const olderPost = 'href="/updates/member-data-export/"';
  assert.ok(listing.content.includes(meeting), 'post-meeting update must be discoverable');
  assert.ok(listing.content.includes(results), 'final survey results must be discoverable');
  assert.ok(listing.content.includes(reminder), 'Nunjucks survey update must be discoverable');
  assert.ok(listing.content.includes(olderPost), 'Markdown updates must remain listed');
  assert.ok(listing.content.indexOf(meeting) < listing.content.indexOf(results), 'post-meeting update must be newest');
  assert.ok(listing.content.indexOf(results) < listing.content.indexOf(reminder), 'final results must be the newest survey update');
  assert.ok(listing.content.indexOf(reminder) < listing.content.indexOf(olderPost), 'newer update must appear first');
  assert.ok(pages.some((page) => page.url === '/updates/after-our-first-jlr-meeting/'), 'post-meeting update must have a page');
  assert.ok(pages.some((page) => page.url === '/updates/survey-final-straight/'), 'listed update must also have a page');
  assert.ok(pages.some((page) => page.url === '/updates/september-survey-results/'), 'final results page must be generated');
});
