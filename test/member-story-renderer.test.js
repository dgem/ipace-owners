'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');
var vm = require('node:vm');

function renderer() {
  var context = {
    document: {
      createElement: function () {
        var value = '';
        return {
          set textContent(text) { value = String(text); },
          get innerHTML() { return value; },
        };
      },
    },
    window: {},
  };
  vm.runInNewContext(fs.readFileSync(path.join(__dirname, '..', 'src/assets/js/story-renderer.js'), 'utf8'), context);
  return context.window.ipaceStoryRenderer;
}

test('member-story media already placed in Markdown is not repeated in the gallery', function () {
  var output = renderer().mediaGallery([
    { id: 'inline', name: 'In place.jpg', url: 'https://example.test/inline.jpg', contentType: 'image/jpeg' },
    { id: 'extra', name: 'Additional.mp4', url: 'https://example.test/extra.mp4', contentType: 'video/mp4' },
  ], 'Opening paragraph.\n\n![In place](story-media:inline)');

  assert.doesNotMatch(output, /In place\.jpg/);
  assert.match(output, /Additional attached media/);
  assert.match(output, /Additional\.mp4/);
});

test('member-story media gallery is absent when every attachment is placed in Markdown', function () {
  var output = renderer().mediaGallery([
    { id: 'inline', name: 'In place.jpg', url: 'https://example.test/inline.jpg', contentType: 'image/jpeg' },
  ], '![In place](story-media:inline)');

  assert.equal(output, '');
});
