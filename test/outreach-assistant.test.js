const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');
const vm = require('node:vm');

const root = path.join(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'src/assets/js/outreach-assistant.js'), 'utf8');

function loadAssistant() {
  const window = {};
  const document = { readyState: 'loading', addEventListener: function () {} };
  const context = { window, document, URL, navigator: {} };
  vm.runInNewContext(source, context);
  return window.ipaceOutreachAssistant;
}

test('generates encoded Facebook links without retrieving them', function () {
  const assistant = loadAssistant();
  const result = assistant.buildSearchLinks(
    'https://www.facebook.com/groups/ipace.owners/posts/123?comment_id=456',
    '"traction battery fault"\nI-PACE H484',
    true
  );

  assert.equal(result.invalidGroupCount, 0);
  assert.equal(result.links.length, 4);
  assert.equal(
    result.links[0].url,
    'https://www.facebook.com/groups/ipace.owners/search/?q=%22traction%20battery%20fault%22'
  );
  assert.equal(
    result.links[1].url,
    'https://www.facebook.com/search/posts/?q=%22traction%20battery%20fault%22'
  );
  assert.doesNotMatch(source, /\bfetch\s*\(|XMLHttpRequest|window\.open|\.submit\s*\(/);
});

test('rejects non-group and non-Facebook destinations', function () {
  const assistant = loadAssistant();
  assert.equal(assistant.normaliseGroupUrl('https://example.com/groups/ipace'), '');
  assert.equal(assistant.normaliseGroupUrl('https://www.facebook.com/profile.php?id=1'), '');
  assert.equal(assistant.normaliseGroupUrl('http://www.facebook.com/groups/ipace'), '');
  assert.equal(assistant.normaliseGroupUrl('https://m.facebook.com/groups/ipace'), 'https://www.facebook.com/groups/ipace');
});

test('drafts useful disclosed replies without claiming diagnosis', function () {
  const assistant = loadAssistant();
  const invited = assistant.draftReply('module-repair', true);
  const notInvited = assistant.draftReply('module-repair', false);

  assert.match(invited, /which module or cells/i);
  assert.match(invited, /I volunteer with/);
  assert.match(invited, /rather than a diagnosis or official Jaguar advice/);
  assert.equal(notInvited.includes('I volunteer with'), false);
});

test('admin outreach page is gated and documents the manual boundary', function () {
  const page = fs.readFileSync(path.join(root, 'src/admin/outreach.njk'), 'utf8');
  const layout = fs.readFileSync(path.join(root, 'src/_includes/layouts/base.njk'), 'utf8');

  assert.match(page, /data-admin-container/);
  assert.match(page, /data-admin-content hidden/);
  assert.match(page, /never reads Facebook/i);
  assert.match(layout, /outreachAssistant[\s\S]*outreach-assistant\.js/);
});
