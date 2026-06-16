'use strict';

var utils = require('./lib/submission-utils');

var COUNTRIES = [
  'GB',
  'IE',
  'DE',
  'FR',
  'NL',
  'NO',
  'SE',
  'DK',
  'AT',
  'CH',
  'BE',
  'ES',
  'IT',
  'PT',
  'US',
  'CA',
  'AU',
  'NZ',
  'other',
];

var MODEL_YEARS = ['2018', '2019', '2020', '2021', '2022', '2023', '2024'];
var SOH_SOURCES = ['dealer-report', 'diagnostic-app', 'service-paperwork', 'jlr-communication', 'estimate-unsure'];

function normaliseVin(value) {
  return String(value || '').toUpperCase().replace(/\s+/g, '').slice(0, 17);
}

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';
  var corsHeaders = utils.corsHeaders(origin);

  utils.log('submit-vehicle-basics', 'info', 'request received', utils.requestMetadata(event, context, {
    hasBody: !!event.body,
  }));

  if (origin && !utils.originAllowed(origin)) {
    utils.log('submit-vehicle-basics', 'warn', 'request rejected: disallowed origin', utils.requestMetadata(event, context));
    return utils.json(403, { error: 'Forbidden' }, corsHeaders);
  }

  if (event.httpMethod === 'OPTIONS') {
    return { statusCode: 204, headers: corsHeaders, body: '' };
  }

  if (event.httpMethod !== 'POST') {
    return utils.json(405, { error: 'Method Not Allowed' }, corsHeaders);
  }

  var user = utils.requireUser(context);
  if (!user || !user.sub) {
    return utils.json(401, { error: 'Sign in required' }, corsHeaders);
  }

  var body;
  try {
    body = utils.parseBody(event);
  } catch (_) {
    return utils.json(400, { error: 'Invalid request body' }, corsHeaders);
  }

  var vin = normaliseVin(body.vin);
  var userHash = utils.emailFingerprint(user.sub);
  if (vin && !/^[A-HJ-NPR-Z0-9]{17}$/.test(vin)) {
    return utils.json(400, { error: 'VIN must be 17 characters and cannot contain I, O, or Q' }, corsHeaders);
  }

  var modelYear = utils.cleanEnum(String(body.modelYear || ''), MODEL_YEARS);
  var country = utils.cleanEnum(body.country, COUNTRIES);
  var registration = utils.cleanString(body.registration, 40).toUpperCase();
  var mileage = utils.cleanInteger(body.mileage, 0, 500000);
  var soh = utils.cleanDecimal(body.soh, 0, 100);
  var sohMileage = utils.cleanInteger(body.sohMileage, 0, 500000);
  var now = new Date().toISOString();
  var id = utils.submissionId('vehicle');

  if (!vin && !registration) {
    return utils.json(400, { error: 'VIN or registration is required' }, corsHeaders);
  }

  var canStoreVinIdentifier = !!(vin && process.env.VIN_PEPPER);
  if (vin && !canStoreVinIdentifier) {
    utils.log('submit-vehicle-basics', registration ? 'warn' : 'error', 'VIN pepper missing', utils.requestMetadata(event, context, {
      userHash: userHash,
      hasRegistration: !!registration,
    }));

    if (!registration) {
      return utils.json(500, { error: 'Vehicle identifier storage is not configured. Provide registration instead, or try again later.' }, corsHeaders);
    }
  }

  var record = {
    id: id,
    type: 'vehicle-basics',
    createdAt: now,
    updatedAt: now,
    identityUserId: user.sub,
    userEmailHash: user.email ? utils.emailFingerprint(String(user.email).toLowerCase()) : null,
    vehicle: {
      vinHash: canStoreVinIdentifier ? utils.hmac(vin, process.env.VIN_PEPPER) : null,
      vinLast6: canStoreVinIdentifier ? vin.slice(-6) : '',
      registration: registration,
      country: country,
      modelYear: modelYear,
      mileage: mileage,
      ownedSince: utils.cleanDate(body.ownedSince),
      firstRegistrationDate: utils.cleanDate(body.firstReg),
    },
    battery: {
      stateOfHealth: soh,
      measuredAt: utils.cleanDate(body.sohDate),
      mileageAtMeasurement: sohMileage,
      source: utils.cleanEnum(body.sohSource, SOH_SOURCES),
    },
    review: {
      status: 'new',
      verificationLevel: 'self-reported',
    },
  };

  try {
    await utils.saveRecord(event, 'vehicle-basics/' + user.sub + '/' + id + '.json', record, {
      type: 'vehicle-basics',
      createdAt: now,
      identityUserId: user.sub,
      status: 'new',
      hasVin: canStoreVinIdentifier,
      modelYear: modelYear || 'unknown',
      country: country || 'unknown',
    });
  } catch (error) {
    utils.log('submit-vehicle-basics', 'error', 'record save failed', utils.requestMetadata(event, context, {
      userHash: userHash,
      errorName: error && error.name,
      errorMessage: error && error.message,
    }));
    return utils.json(500, { ok: false, error: 'Could not save vehicle basics' }, corsHeaders);
  }

  utils.log('submit-vehicle-basics', 'info', 'record saved', utils.requestMetadata(event, context, {
    userHash: userHash,
    submissionId: id,
    hasVin: canStoreVinIdentifier,
  }));

  return utils.json(200, { ok: true, id: id }, corsHeaders);
};
