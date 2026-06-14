'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var utils = require('../netlify/functions/lib/submission-utils');

test('originAllowed accepts production, deploy previews, and localhost only', function () {
  assert.equal(utils.originAllowed('https://ipace-owners.org'), true);
  assert.equal(utils.originAllowed('https://feature-x--ipace-owners.netlify.app'), true);
  assert.equal(utils.originAllowed('http://localhost:8888'), true);
  assert.equal(utils.originAllowed('https://evil.example'), false);
  assert.equal(utils.originAllowed(''), false);
});

test('parseBody supports JSON and repeated urlencoded fields', function () {
  assert.deepEqual(
    utils.parseBody({
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ email: 'User@Example.COM' }),
    }),
    { email: 'User@Example.COM' }
  );

  assert.deepEqual(
    utils.parseBody({
      headers: { 'content-type': 'application/x-www-form-urlencoded' },
      body: 'skills=legal&skills=data&email=user%40example.com',
    }),
    { skills: ['legal', 'data'], email: 'user@example.com' }
  );
});

test('cleaning helpers constrain values conservatively', function () {
  assert.equal(utils.cleanEmail(' USER@Example.COM '), 'user@example.com');
  assert.equal(utils.isEmail('user@example.com'), true);
  assert.equal(utils.isEmail('not-an-email'), false);
  assert.equal(utils.cleanEnum('bad', ['good']), '');
  assert.equal(utils.cleanDate('2026-06-14'), '2026-06-14');
  assert.equal(utils.cleanDate('14/06/2026'), '');
  assert.equal(utils.cleanInteger('42', 0, 100), 42);
  assert.equal(utils.cleanInteger('42.5', 0, 100), null);
  assert.equal(utils.cleanDecimal('92.46', 0, 100), 92.5);
});
