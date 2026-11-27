const fs = require('node:fs');
const test = require('node:test');
const assert = require('node:assert/strict');

test('owners story navigation and content start hidden pending role resolution', function () {
  const nav = JSON.parse(fs.readFileSync('src/_data/navigation.json', 'utf8'));
  assert.equal(nav.public.find((item) => item.url === '/member-stories/').feature, 'owners-stories');
  for (const template of ['src/_includes/partials/header.njk', 'src/_includes/partials/mobile-nav.njk']) {
    assert.match(fs.readFileSync(template, 'utf8'), /data-member-feature="\{\{ item.feature \}\}" hidden/);
  }
  const page = fs.readFileSync('src/member-stories.njk', 'utf8');
  assert.match(page, /data-member-feature="owners-stories" hidden/);
  assert.match(page, /noindex: true/);
  assert.match(page, /data-member-feature-denied="owners-stories" hidden>[\s\S]*?Page Not Found/);
  assert.doesNotMatch(page, /currently available to approved early adopters/);
});
