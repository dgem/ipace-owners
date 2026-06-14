'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var utils = require('../netlify/functions/lib/submission-utils');
var identityMagicLink = require('../netlify/functions/lib/identity-magic-link');
var submitJoin = require('../netlify/functions/submit-join');

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

test('submit-join stores the membership record and sends one server-side magic link for guests', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var originalSendMagicLink = identityMagicLink.sendMagicLink;
  var saved = null;
  var magicLinkCalls = [];

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
    identityMagicLink.sendMagicLink = originalSendMagicLink;
  });

  utils.saveRecord = async function (_event, key, record, metadata) {
    saved = { key: key, record: record, metadata: metadata };
  };
  identityMagicLink.sendMagicLink = async function (options) {
    magicLinkCalls.push(options);
    return { ok: true };
  };

  var res = await submitJoin.handler(event({
    name: 'Test User',
    email: 'Test@Example.COM',
    country: 'GB',
    relationship: 'current-owner',
    ownership: 'one',
    skills: ['legal', 'data'],
    'consent-contact': 'yes',
    'consent-not-legal': 'yes',
    'consent-data': 'yes',
  }), {});
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.ok, true);
  assert.equal(body.magicLinkSent, true);
  assert.equal(body.signedIn, false);
  assert.equal(magicLinkCalls.length, 1);
  assert.equal(magicLinkCalls[0].email, 'test@example.com');
  assert.equal(magicLinkCalls[0].name, 'Test User');

  assert.ok(saved.key.startsWith('join/join_'));
  assert.equal(saved.record.contact.email, 'test@example.com');
  assert.deepEqual(saved.record.membership.skills, ['legal', 'data']);
  assert.equal(saved.record.consents.contact, true);
  assert.equal(saved.record.consents.notLegalClaim, true);
  assert.equal(saved.record.consents.anonymisedAnalysis, true);
  assert.equal(saved.metadata.type, 'join');
  assert.equal(typeof saved.metadata.emailHash, 'string');
});

test('submit-join does not send a magic link when the user is already signed in', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var originalSendMagicLink = identityMagicLink.sendMagicLink;
  var magicLinkCalls = 0;

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
    identityMagicLink.sendMagicLink = originalSendMagicLink;
  });

  utils.saveRecord = async function () {};
  identityMagicLink.sendMagicLink = async function () {
    magicLinkCalls += 1;
    return { ok: true };
  };

  var res = await submitJoin.handler(event({
    name: 'Signed In User',
    email: 'member@example.com',
    'consent-contact': 'yes',
    'consent-not-legal': 'yes',
  }), {
    clientContext: {
      user: { sub: 'identity-user-1', email: 'member@example.com' },
    },
  });
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.ok, true);
  assert.equal(body.magicLinkSent, true);
  assert.equal(body.signedIn, true);
  assert.equal(magicLinkCalls, 0);
});

test('submit-join rejects invalid email before saving or sending email', async function (t) {
  var originalSaveRecord = utils.saveRecord;
  var originalSendMagicLink = identityMagicLink.sendMagicLink;
  var saveCalls = 0;
  var magicLinkCalls = 0;

  t.after(function () {
    utils.saveRecord = originalSaveRecord;
    identityMagicLink.sendMagicLink = originalSendMagicLink;
  });

  utils.saveRecord = async function () { saveCalls += 1; };
  identityMagicLink.sendMagicLink = async function () { magicLinkCalls += 1; };

  var res = await submitJoin.handler(event({
    name: 'Test User',
    email: 'not-an-email',
    'consent-contact': 'yes',
    'consent-not-legal': 'yes',
  }), {});

  assert.equal(res.statusCode, 400);
  assert.equal(saveCalls, 0);
  assert.equal(magicLinkCalls, 0);
});
