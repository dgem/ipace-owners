'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var utils = require('../netlify/functions/lib/submission-utils');
var submitVehicleBasics = require('../netlify/functions/submit-vehicle-basics');

function event(body) {
  return {
    httpMethod: 'POST',
    headers: {
      origin: 'http://localhost:8888',
      'content-type': 'application/json',
    },
    body: JSON.stringify(body),
  };
}

var context = {
  clientContext: {
    user: { sub: 'identity-user-1', email: 'owner@example.com' },
  },
};

test('submit-vehicle-basics requires a signed-in user', async function () {
  var res = await submitVehicleBasics.handler(event({ registration: 'AB12 CDE' }), {});

  assert.equal(res.statusCode, 401);
  assert.deepEqual(JSON.parse(res.body), { error: 'Sign in required' });
});

test('submit-vehicle-basics requires VIN or registration', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var saveCalls = 0;

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
  });

  utils.saveRecord = async function () { saveCalls += 1; };

  var res = await submitVehicleBasics.handler(event({ country: 'GB' }), context);

  assert.equal(res.statusCode, 400);
  assert.equal(JSON.parse(res.body).error, 'VIN or registration is required');
  assert.equal(saveCalls, 0);
});

test('submit-vehicle-basics stores VIN HMAC and never stores the full VIN', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var originalVinPepper = process.env.VIN_PEPPER;
  var saved = null;

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
    if (originalVinPepper === undefined) {
      delete process.env.VIN_PEPPER;
    } else {
      process.env.VIN_PEPPER = originalVinPepper;
    }
  });

  process.env.VIN_PEPPER = 'test-pepper';
  utils.saveRecord = async function (_event, key, record, metadata) {
    saved = { key: key, record: record, metadata: metadata };
  };

  var vin = 'SADHB2S10K1F12345';
  var res = await submitVehicleBasics.handler(event({
    vin: vin,
    registration: 'ab12 cde',
    country: 'GB',
    modelYear: '2019',
    mileage: '42000',
    ownedSince: '2024-01-02',
    firstReg: '2019-03-04',
    soh: '91.26',
    sohDate: '2026-06-14',
    sohMileage: '41000',
    sohSource: 'dealer-report',
  }), context);
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.ok, true);
  assert.ok(saved.key.startsWith('vehicle-basics/identity-user-1/vehicle_'));
  assert.equal(saved.record.identityUserId, 'identity-user-1');
  assert.equal(saved.record.vehicle.vinLast6, 'F12345');
  assert.equal(saved.record.vehicle.registration, 'AB12 CDE');
  assert.equal(saved.record.vehicle.country, 'GB');
  assert.equal(saved.record.vehicle.modelYear, '2019');
  assert.equal(saved.record.vehicle.mileage, 42000);
  assert.equal(saved.record.battery.stateOfHealth, 91.3);
  assert.equal(saved.record.battery.source, 'dealer-report');
  assert.equal(saved.metadata.hasVin, true);

  var serialized = JSON.stringify(saved.record);
  assert.equal(serialized.includes(vin), false);
  assert.equal(typeof saved.record.vehicle.vinHash, 'string');
  assert.equal(saved.record.vehicle.vinHash.length, 64);
});

test('submit-vehicle-basics rejects VIN writes when VIN_PEPPER is missing', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var originalVinPepper = process.env.VIN_PEPPER;
  var saveCalls = 0;

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
    if (originalVinPepper === undefined) {
      delete process.env.VIN_PEPPER;
    } else {
      process.env.VIN_PEPPER = originalVinPepper;
    }
  });

  delete process.env.VIN_PEPPER;
  utils.saveRecord = async function () { saveCalls += 1; };

  var res = await submitVehicleBasics.handler(event({ vin: 'SADHB2S10K1F12345' }), context);

  assert.equal(res.statusCode, 500);
  assert.equal(JSON.parse(res.body).error, 'Vehicle identifier storage is not configured');
  assert.equal(saveCalls, 0);
});
