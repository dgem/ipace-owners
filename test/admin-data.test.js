'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var adminData = require('../netlify/functions/admin-data');
var utils = require('../netlify/functions/lib/submission-utils');

// ── Mock blobs for all tests in this file ──────────────────────────────────────
function createMockStore(name) {
  var data = {};
  return {
    name: name,
    setJSON: async function (key, value, opts) {
      data[key] = { value: value, metadata: opts && opts.metadata || {} };
      },
    get: async function (key, opts) {
      if (!opts || opts.type !== 'json') return null;
      return data[key] && data[key].value || null;
      },
    list: async function (options) {
      var prefix = typeof options === 'string' ? options : options && options.prefix;
      var keys = Object.keys(data).filter(function (k) {
        return !prefix || k.startsWith(prefix);
        });
      return { blobs: keys.map(function (key) {
        return {
          key: key,
           };
          }) };
      },
     _setData: function (key, value, metadata) {
      data[key] = { value: value, metadata: metadata || {} };
      },
    };
}

test.afterEach(function () {
  // Reset mock stores between tests
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

test('admin-data rejects unauthenticated requests', async function () {
  var res = await adminData.handler(event('GET'), {});
  assert.equal(res.statusCode, 401);
  assert.deepEqual(JSON.parse(res.body), { error: 'Sign in required' });
});

test('admin-data rejects authenticated non-admin users', async function () {
  var res = await adminData.handler(
    event('GET'),
    contextWithUser({ sub: 'user-123', email: 'member@example.com', app_metadata: {} })
   );
  assert.equal(res.statusCode, 403);
  assert.deepEqual(JSON.parse(res.body), { error: 'Admin access required' });
});

test('admin-data rejects users without admin role in app_metadata.roles', async function () {
  var res = await adminData.handler(
    event('GET'),
    contextWithUser({ sub: 'user-123', email: 'member@example.com', app_metadata: { roles: ['member'] } })
   );
  assert.equal(res.statusCode, 403);
  assert.deepEqual(JSON.parse(res.body), { error: 'Admin access required' });
});

test('admin-data only allows GET method', async function () {
  var res = await adminData.handler(
    event('POST'),
    contextWithUser({ sub: 'admin-1', email: 'admin@example.com', app_metadata: { roles: ['admin'] } })
   );
  assert.equal(res.statusCode, 405);
  assert.deepEqual(JSON.parse(res.body), { error: 'Method Not Allowed' });
});

test('admin-data handles CORS preflight', async function () {
  var res = await adminData.handler(event('OPTIONS'), {});
  assert.equal(res.statusCode, 204);
});

test('admin-data rejects disallowed origins even for admins', async function () {
  var ev = event('GET');
  ev.headers.origin = 'https://evil.example';
  var res = await adminData.handler(
    ev,
    contextWithUser({ sub: 'admin-1', email: 'admin@example.com', app_metadata: { roles: ['admin'] } })
   );
  assert.equal(res.statusCode, 403);
  assert.deepEqual(JSON.parse(res.body), { error: 'Forbidden' });
});

test('admin-data returns all records for admin users', async function (t) {
  var mockStore = createMockStore('owner-submissions');
  mockStore._setData('join/join_abc.json', {
    id: 'join_abc',
    type: 'join',
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    identityUserId: 'user-123',
    contact: { name: 'User One', email: 'one@example.com' },
    membership: { relationship: 'current-owner' },
    consents: { contact: true },
    review: { status: 'new' },
     });
  mockStore._setData('join/join_xyz.json', {
    id: 'join_xyz',
    type: 'join',
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
    identityUserId: 'user-456',
    contact: { name: 'User Two', email: 'two@example.com' },
    membership: { relationship: 'former-owner' },
    consents: { contact: true, anonymisedAnalysis: true },
    review: { status: 'verified' },
     });
  mockStore._setData('vehicle-basics/user-123/vehicle_v1.json', {
    id: 'vehicle_v1',
    type: 'vehicle-basics',
    createdAt: '2026-03-01T00:00:00Z',
    updatedAt: '2026-03-01T00:00:00Z',
    identityUserId: 'user-123',
    userEmailHash: 'hash123',
    vehicle: { registration: 'AB12 CDE', country: 'GB' },
    battery: { stateOfHealth: 91.3 },
    review: { status: 'new' },
     });
  mockStore._setData('vehicle-basics/user-456/vehicle_v2.json', {
    id: 'vehicle_v2',
    type: 'vehicle-basics',
    createdAt: '2026-04-01T00:00:00Z',
    updatedAt: '2026-04-01T00:00:00Z',
    identityUserId: 'user-456',
    userEmailHash: 'hash456',
    vehicle: { registration: 'XY99 ZZZ', country: 'DE' },
    battery: { stateOfHealth: 85.0 },
    review: { status: 'verified' },
     });

  var originalGetStore = utils.getStore;
  t.after(function () {
    utils.getStore = originalGetStore;
   });

  utils.getStore = function () { return mockStore; };

  var res = await adminData.handler(
    event('GET'),
    contextWithUser({ sub: 'admin-1', email: 'admin@example.com', app_metadata: { roles: ['admin'] } })
   );
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.identityUserId, 'admin-1');
  assert.equal(body.email, 'admin@example.com');

    // Admin sees ALL join records
  assert.equal(body.joinRecords.length, 2);
  assert.equal(body.joinRecords[0].id, 'join_abc');
  assert.equal(body.joinRecords[1].id, 'join_xyz');

    // Admin sees review data for join records
  body.joinRecords.forEach(function (rec) {
    assert.ok(rec.review);
    assert.ok(rec.identityUserId);
    });

    // Admin sees ALL vehicle records
  assert.equal(body.vehicleRecords.length, 2);
  assert.equal(body.vehicleRecords[0].id, 'vehicle_v1');
  assert.equal(body.vehicleRecords[1].id, 'vehicle_v2');

    // Admin sees userEmailHash and review data for vehicle records
  body.vehicleRecords.forEach(function (rec) {
    assert.ok(rec.userEmailHash);
    assert.ok(rec.review);
    assert.ok(rec.identityUserId);
    });
});

test('admin-data returns empty arrays when no records exist', async function (t) {
  var mockStore = createMockStore('owner-submissions');

  var originalGetStore = utils.getStore;
  t.after(function () {
    utils.getStore = originalGetStore;
   });

  utils.getStore = function () { return mockStore; };

  var res = await adminData.handler(
    event('GET'),
    contextWithUser({ sub: 'admin-1', email: 'admin@example.com', app_metadata: { roles: ['admin'] } })
   );
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.joinRecords.length, 0);
  assert.equal(body.vehicleRecords.length, 0);
});
