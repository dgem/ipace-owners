'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var ownerData = require('../netlify/functions/lib/owner-data');
var utils = require('../netlify/functions/lib/submission-utils');

function createSqlMock(fixtures) {
  var calls = [];

  async function sql(strings) {
    var values = Array.prototype.slice.call(arguments, 1);
    var text = Array.prototype.join.call(strings, '?');
    calls.push({ text: text, values: values });

    if (/INSERT INTO members/.test(text)) {
      return [{ id: fixtures.memberId || 'member_1' }];
    }

    if (/SELECT id, identity_user_id\s+FROM members/.test(text)) {
      return fixtures.existingMember ? [fixtures.existingMember] : [];
    }

    if (/SELECT contact_name/.test(text)) {
      return fixtures.latestJoin ? [fixtures.latestJoin] : [];
    }

    if (/FROM join_submissions/.test(text) && /ORDER BY created_at DESC/.test(text)) {
      return fixtures.joinRows || [];
    }

    if (/FROM vehicles v/.test(text)) {
      return fixtures.vehicleRows || [];
    }

    return [];
  }

  return {
    db: { sql: sql },
    calls: calls,
  };
}

function createStore() {
  var data = {};
  return {
    setJSON: async function (key, value, opts) {
      data[key] = { value: value, metadata: opts && opts.metadata };
    },
    get: async function (key, opts) {
      if (!opts || opts.type !== 'json') return null;
      return data[key] && data[key].value || null;
    },
    data: data,
  };
}

test('saveVehicleRecord writes canonical database rows and refreshes private member snapshot', async function (t) {
  var originalGetDatabaseConnection = ownerData.getDatabaseConnection;
  var originalGetStore = utils.getStore;
  var store = createStore();
  var sqlMock = createSqlMock({
    memberId: 'member_1',
    existingMember: { id: 'member_1', identity_user_id: 'identity-user-1' },
    vehicleRows: [{
      id: 'vehicle_1',
      identity_user_id: 'identity-user-1',
      registration: 'AB12 CDE',
      country: 'GB',
      model_year: '2019',
      current_mileage: 42000,
      created_at: '2026-06-16T00:00:00.000Z',
      updated_at: '2026-06-16T00:00:00.000Z',
      state_of_health: 91.2,
      source: 'dealer-report',
    }],
  });

  t.after(function () {
    ownerData.getDatabaseConnection = originalGetDatabaseConnection;
    utils.getStore = originalGetStore;
  });

  ownerData.getDatabaseConnection = function () { return sqlMock.db; };
  utils.getStore = function () { return store; };

  var saved = await ownerData.saveVehicleRecord({}, {
    id: 'vehicle_1',
    createdAt: '2026-06-16T00:00:00.000Z',
    updatedAt: '2026-06-16T00:00:00.000Z',
    identityUserId: 'identity-user-1',
    userEmailHash: 'emailhash',
    vehicle: {
      vinHash: 'vinhash',
      vinLast6: 'F12345',
      registration: 'AB12 CDE',
      country: 'GB',
      modelYear: '2019',
      mileage: 42000,
    },
    battery: {
      stateOfHealth: 91.2,
      source: 'dealer-report',
    },
    review: {
      status: 'new',
      verificationLevel: 'self-reported',
    },
  });

  assert.equal(saved, true);
  assert.equal(sqlMock.calls.some(function (call) { return /INSERT INTO vehicles/.test(call.text); }), true);
  assert.equal(sqlMock.calls.some(function (call) { return /INSERT INTO vehicle_battery_readings/.test(call.text); }), true);
  assert.equal(sqlMock.calls.some(function (call) { return /INSERT INTO member_static_snapshots/.test(call.text); }), true);
  assert.equal(store.data['member-snapshots/identity-user-1.json'].value.vehicleRecords.length, 1);
  assert.equal(store.data['member-snapshots/identity-user-1.json'].value.vehicleRecords[0].vehicle.registration, 'AB12 CDE');
});

test('getMemberSnapshot reads the generated private snapshot before regenerating', async function (t) {
  var originalGetDatabaseConnection = ownerData.getDatabaseConnection;
  var originalRegenerateMemberSnapshot = ownerData.regenerateMemberSnapshot;
  var originalGetStore = utils.getStore;
  var store = createStore();
  var regenerateCalls = 0;

  t.after(function () {
    ownerData.getDatabaseConnection = originalGetDatabaseConnection;
    ownerData.regenerateMemberSnapshot = originalRegenerateMemberSnapshot;
    utils.getStore = originalGetStore;
  });

  store.data['member-snapshots/identity-user-1.json'] = {
    value: {
      identityUserId: 'identity-user-1',
      joinRecords: [],
      vehicleRecords: [{ id: 'vehicle_1' }],
    },
  };

  utils.getStore = function () { return store; };
  ownerData.regenerateMemberSnapshot = async function () {
    regenerateCalls += 1;
    return null;
  };

  var snapshot = await ownerData.getMemberSnapshot({}, {
    sub: 'identity-user-1',
    email: 'owner@example.com',
  });

  assert.equal(snapshot.vehicleRecords.length, 1);
  assert.equal(regenerateCalls, 0);
});

test('getAdminData reads review-capable records from Postgres', async function (t) {
  var originalGetDatabaseConnection = ownerData.getDatabaseConnection;
  var sqlMock = createSqlMock({
    joinRows: [{
      id: 'join_1',
      identity_user_id: 'identity-user-1',
      email_hash: 'emailhash',
      contact_name: 'Owner',
      contact_country: 'GB',
      relationship_status: 'current-owner-one',
      skills: ['data'],
      consents: { contact: true },
      review_status: 'new',
      verification_level: 'self-reported',
      created_at: '2026-06-16T00:00:00.000Z',
      updated_at: '2026-06-16T00:00:00.000Z',
    }],
    vehicleRows: [{
      id: 'vehicle_1',
      identity_user_id: 'identity-user-1',
      user_email_hash: 'emailhash',
      registration: 'AB12 CDE',
      review_status: 'new',
      verification_level: 'self-reported',
      created_at: '2026-06-16T00:00:00.000Z',
      updated_at: '2026-06-16T00:00:00.000Z',
    }],
  });

  t.after(function () {
    ownerData.getDatabaseConnection = originalGetDatabaseConnection;
  });

  ownerData.getDatabaseConnection = function () { return sqlMock.db; };

  var data = await ownerData.getAdminData();

  assert.equal(data.joinRecords.length, 1);
  assert.equal(data.joinRecords[0].userEmailHash, 'emailhash');
  assert.equal(data.joinRecords[0].review.status, 'new');
  assert.equal(data.vehicleRecords.length, 1);
  assert.equal(data.vehicleRecords[0].vehicle.registration, 'AB12 CDE');
  assert.equal(data.vehicleRecords[0].review.verificationLevel, 'self-reported');
});
