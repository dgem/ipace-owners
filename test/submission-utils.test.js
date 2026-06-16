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

test('storage helpers export the default store adapter used in production', function () {
  assert.equal(typeof utils.getStore, 'function');
  assert.equal(typeof utils.saveRecord, 'function');
  assert.equal(typeof utils.listJsonRecords, 'function');
});

test('listJsonRecords reads Netlify Blob list results by key', async function (t) {
  var originalGetStore = utils.getStore;
  var data = {
    'vehicle-basics/user-1/vehicle_1.json': { id: 'vehicle_1' },
    'vehicle-basics/user-2/vehicle_2.json': { id: 'vehicle_2' },
  };

  t.after(function () {
    utils.getStore = originalGetStore;
  });

  utils.getStore = function () {
    return {
      list: async function (options) {
        var prefix = options && options.prefix;
        return {
          blobs: Object.keys(data).filter(function (key) {
            return key.startsWith(prefix);
          }).map(function (key) {
            return { key: key };
          }),
        };
      },
      get: async function (key, options) {
        assert.equal(options.type, 'json');
        return data[key] || null;
      },
    };
  };

  assert.deepEqual(
    await utils.listJsonRecords({}, 'vehicle-basics/user-1/'),
    [{ id: 'vehicle_1' }]
  );
});
