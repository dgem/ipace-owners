'use strict';

var assert = require('node:assert/strict');
var fs = require('node:fs');
var path = require('node:path');
var test = require('node:test');

var root = path.join(__dirname, '..');

test('Greece is available wherever members choose a country', function () {
  var sources = [
    'src/join.njk',
    'src/member/submit-vehicle-data.njk',
    'src/assets/js/member-auth.js'
  ];

  for (var index = 0; index < sources.length; index += 1) {
    var source = fs.readFileSync(path.join(root, sources[index]), 'utf8');
    assert.match(source, /value=["']GR["']>Greece<\/option>/, sources[index]);
  }
});
