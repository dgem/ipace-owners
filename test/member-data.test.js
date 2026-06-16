'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var memberData = require('../netlify/functions/member-data');
var utils = require('../netlify/functions/lib/submission-utils');

// ── Mock blobs for all tests in this file ──────────────────────────────────────
var mockStores = {};

function createMockStore(name) {
  var data = {};
  return {
    name: name,
    setJSON: async function (key, value, opts) {
      data[key] = { value: value, metadata: opts && opts.metadata || {} };
     },
    getJSON: async function () {
      // Called on a "key object" — we simulate via the key path stored
      return null;
     },
    list: async function (prefix) {
      var keys = Object.keys(data).filter(function (k) {
        return !prefix || k.startsWith(prefix);
       });
      return keys.map(function (key) {
        return {
          key: key,
          getJSON: async function () {
            return data[key] && data[key].value || null;
           },
          };
         });
     },
    _setData: function (key, value, metadata) {
      data[key] = { value: value, metadata: metadata || {} };
     },
    _data: data,
   };
}

test.afterEach(function () {
  mockStores = {};
});

function event(method) {
  return {
    httpMethod: method || 'GET',
    headers: {
      origin: 'http://localhost:8888',
       'content-type': 'application/json',
     },
    body: '',
   };
}

function contextWithUser(userOpts) {
  return {
    clientContext: {
      user: userOpts || { sub: 'user-123', email: 'test@example.com' },
     },
   };
}

// ── Tests ──────────────────────────────────────────────────────────────────────

test('member-data rejects unauthenticated requests', async function () {
  var res = await memberData.handler(event('GET'), {});
  assert.equal(res.statusCode, 401);
  assert.deepEqual(JSON.parse(res.body), { error: 'Sign in required' });
});

test('member-data only allows GET method', async function () {
  var res = await memberData.handler(event('POST'), contextWithUser());
  assert.equal(res.statusCode, 405);
  assert.deepEqual(JSON.parse(res.body), { error: 'Method Not Allowed' });
});

test('member-data handles CORS preflight', async function () {
  var res = await memberData.handler(event('OPTIONS'), {});
  assert.equal(res.statusCode, 204);
});

test('member-data rejects disallowed origins', async function () {
  var ev = event('GET');
  ev.headers.origin = 'https://evil.example';
  var res = await memberData.handler(ev, contextWithUser());
  assert.equal(res.statusCode, 403);
  assert.deepEqual(JSON.parse(res.body), { error: 'Forbidden' });
});

test('member-data returns user join and vehicle records', async function (t) {
  // Mock blobs module
  var mockStore = createMockStore('owner-submissions');
  mockStore._setData('join/join_abc.json', {
    id: 'join_abc',
    type: 'join',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    identityUserId: 'user-123',
    contact: { name: 'Test User', email: 'test@example.com' },
    membership: { relationship: 'current-owner' },
    consents: { contact: true },
    });
  mockStore._setData('join/join_xyz.json', {
    id: 'join_xyz',
    type: 'join',
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
    identityUserId: 'other-user',
    contact: { name: 'Other User', email: 'other@example.com' },
    membership: { relationship: 'prospective' },
    consents: { contact: true },
    });
  mockStore._setData('vehicle-basics/user-123/vehicle_v1.json', {
    id: 'vehicle_v1',
    type: 'vehicle-basics',
    createdAt: '2026-03-01T00:00:00Z',
    updatedAt: '2026-03-01T00:00:00Z',
    identityUserId: 'user-123',
    vehicle: { registration: 'AB12 CDE', country: 'GB' },
    battery: { stateOfHealth: 91.3 },
    });
  mockStore._setData('vehicle-basics/other-user/vehicle_v2.json', {
    id: 'vehicle_v2',
    type: 'vehicle-basics',
    createdAt: '2026-04-01T00:00:00Z',
    updatedAt: '2026-04-01T00:00:00Z',
    identityUserId: 'other-user',
    vehicle: { registration: 'XY99 ZZZ', country: 'DE' },
    battery: { stateOfHealth: 85.0 },
    });

  var originalGetStore = utils.getStore;
  t.after(function () {
    utils.getStore = originalGetStore;
   });

  utils.getStore = function () { return mockStore; };

  var res = await memberData.handler(event('GET'), contextWithUser({ sub: 'user-123', email: 'test@example.com' }));
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.identityUserId, 'user-123');
  assert.equal(body.email, 'test@example.com');

   // Should only return user's own join record
  assert.equal(body.joinRecords.length, 1);
  assert.equal(body.joinRecords[0].id, 'join_abc');
  assert.equal(body.joinRecords[0].contact.name, 'Test User');

   // Should not expose the other user's join record
  body.joinRecords.forEach(function (rec) {
    assert.equal(rec.identityUserId, 'user-123');
   });

   // Should only return user's own vehicle record
  assert.equal(body.vehicleRecords.length, 1);
  assert.equal(body.vehicleRecords[0].id, 'vehicle_v1');
  assert.equal(body.vehicleRecords[0].vehicle.registration, 'AB12 CDE');

   // Should not expose the other user's vehicle record
  body.vehicleRecords.forEach(function (rec) {
    assert.equal(rec.identityUserId, 'user-123');
   });
});

test('member-data returns empty arrays when no records exist', async function (t) {
  var mockStore = createMockStore('owner-submissions');

  var originalGetStore = utils.getStore;
  t.after(function () {
    utils.getStore = originalGetStore;
   });

  utils.getStore = function () { return mockStore; };

  var res = await memberData.handler(event('GET'), contextWithUser());
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.joinRecords.length, 0);
  assert.equal(body.vehicleRecords.length, 0);
});

test('member-data does not expose sensitive review data to members', async function (t) {
  var mockStore = createMockStore('owner-submissions');
  mockStore._setData('join/join_abc.json', {
    id: 'join_abc',
    type: 'join',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    identityUserId: 'user-123',
    contact: { name: 'Test User', email: 'test@example.com' },
    membership: { relationship: 'current-owner' },
    consents: { contact: true },
    review: { status: 'verified', verificationLevel: 'confirmed' },
    });

  var originalGetStore = utils.getStore;
  t.after(function () {
    utils.getStore = originalGetStore;
   });

  utils.getStore = function () { return mockStore; };

  var res = await memberData.handler(event('GET'), contextWithUser());
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
   // review field should not be in the response for members
  assert.equal(body.joinRecords[0].review, undefined);
});
