'use strict';
var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');
var root = path.join(__dirname, '..');

test('member story admin workspace is paged, filterable and supports bulk archive', function () {
  var js = fs.readFileSync(path.join(root, 'src/assets/js/admin-member-stories.js'), 'utf8');
  var go = fs.readFileSync(path.join(root, 'functions/firebase-go/member_stories.go'), 'utf8');
  assert.match(js, /pageSize/);
  assert.match(js, /data-select-all/);
  assert.match(js, /action: action, ids/);
  assert.match(js, /admin-story-row__preview/);
  assert.match(js, /admin\/member-stories\/edit\/\?id=/);
  assert.match(js, /Approve selected/);
  assert.match(js, /Publish selected/);
  assert.match(js, /Archive selected/);
  assert.doesNotMatch(js, /data-bulk-action" aria-label/);
  var editor = fs.readFileSync(path.join(root, 'src/assets/js/admin-member-story-edit.js'), 'utf8');
  assert.match(editor, /Share clicks by platform/);
  assert.match(editor, /Story \(Markdown source\)/);
  assert.match(editor, /story-markdown-editor/);
  assert.match(editor, /Sentiment:/);
  assert.match(editor, /findStory\(page \+ 1\)/);
  assert.match(js, /btn--danger btn--sm" data-bulk-action="archive"/);
  assert.match(go, /hasNext/);
  assert.match(go, /query.Get\("status"\)/);
  assert.match(go, /input.Action == "archive" \|\| input.Action == "approve" \|\| input.Action == "publish"/);
});

test('public story sharing provides platform actions and records platform click context', function () {
  var js = fs.readFileSync(path.join(root, 'src/assets/js/public-member-stories.js'), 'utf8');
  var go = fs.readFileSync(path.join(root, 'functions/firebase-go/member_stories.go'), 'utf8');
  for (var platform of ['facebook', 'x', 'whatsapp', 'linkedin', 'instagram', 'tiktok', 'copy']) {
    assert.match(js, new RegExp(platform, 'i'));
  }
  assert.match(js, /navigator\.share/);
  assert.match(js, /platform: platform \|\| ''/);
  assert.match(js, /Owners’ story/);
  assert.match(js, /Read this story/);
  assert.match(js, /An I-PACE owner story/);
  assert.match(go, /SharePlatforms map\[string\]int/);
  assert.match(go, /validStorySharePlatform\(input\.Platform\)/);
});

test('members see published state and can open their published story', function () {
  var js = fs.readFileSync(path.join(root, 'src/assets/js/member-stories.js'), 'utf8');
  assert.match(js, /statusLabel\(story\.status\)/);
  assert.match(js, /story\.status === 'published'/);
  assert.match(js, /View published experience/);
});
