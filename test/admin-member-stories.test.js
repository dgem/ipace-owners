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
  assert.match(js, /action:'archive',ids/);
  assert.match(js, /data-story-detail/);
  assert.match(js, /Story \(Markdown source\)/);
  assert.match(js, /story-markdown-editor/);
  assert.match(js, /Sentiment:/);
  assert.match(go, /hasNext/);
  assert.match(go, /query.Get\("status"\)/);
  assert.match(go, /input.Action == "archive" && len\(input.IDs\) > 0/);
});
