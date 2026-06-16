'use strict';

var utils = require('./lib/submission-utils');
var identityMagicLink = require('./lib/identity-magic-link');

var RELATIONSHIPS = [
  'current-owner-one',
  'current-owner-multiple',
  'former-owner',
  'prospective-buyer',
  'helping-owner',
  'trade-specialist',
  'other',
];
var SKILLS = [
  'legal',
  'technical',
  'data',
  'media',
  'web',
  'consumer-rights',
  'dealer',
  'general',
];

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';
  var corsHeaders = utils.corsHeaders(origin);

  utils.log('submit-join', 'info', 'request received', utils.requestMetadata(event, context, {
    hasBody: !!event.body,
  }));

  if (origin && !utils.originAllowed(origin)) {
    utils.log('submit-join', 'warn', 'request rejected: disallowed origin', utils.requestMetadata(event, context));
    return utils.json(403, { error: 'Forbidden' }, corsHeaders);
  }

  if (event.httpMethod === 'OPTIONS') {
    return { statusCode: 204, headers: corsHeaders, body: '' };
  }

  if (event.httpMethod !== 'POST') {
    return utils.json(405, { error: 'Method Not Allowed' }, corsHeaders);
  }

  var body;
  try {
    body = utils.parseBody(event);
  } catch (_) {
    return utils.json(400, { error: 'Invalid request body' }, corsHeaders);
  }

  if (utils.cleanString(body['bot-field'], 100)) {
    utils.log('submit-join', 'warn', 'honeypot submission ignored', utils.requestMetadata(event, context));
    return utils.json(200, { ok: true }, corsHeaders);
  }

  var email = utils.cleanEmail(body.email);
  var name = utils.cleanString(body.name, 200);

  if (!name) {
    return utils.json(400, { error: 'Name is required' }, corsHeaders);
  }

  if (!email || !utils.isEmail(email)) {
    return utils.json(400, { error: 'Valid email address required' }, corsHeaders);
  }

  if (utils.cleanString(body['consent-contact'], 10) !== 'yes') {
    return utils.json(400, { error: 'Contact consent is required' }, corsHeaders);
  }

  if (utils.cleanString(body['consent-not-legal'], 10) !== 'yes') {
    return utils.json(400, { error: 'Participation acknowledgement is required' }, corsHeaders);
  }

  var now = new Date().toISOString();
  var id = utils.submissionId('join');
  var emailHash = utils.emailFingerprint(email);
  var user = utils.requireUser(context);

  var record = {
    id: id,
    type: 'join',
    createdAt: now,
    updatedAt: now,
    identityUserId: user && user.sub ? user.sub : null,
    contact: {
      name: name,
      email: email,
      country: utils.cleanString(body.country, 80),
    },
    membership: {
      relationship: utils.cleanEnum(body.relationship, RELATIONSHIPS),
      skills: utils.filterEnums(body.skills, SKILLS),
    },
    consents: {
      contact: true,
      notLegalClaim: true,
      anonymisedAnalysis: utils.cleanString(body['consent-data'], 10) === 'yes',
    },
    review: {
      status: 'new',
      verificationLevel: 'self-reported',
    },
  };

  try {
    await utils.saveRecord(event, 'join/' + id + '.json', record, {
      type: 'join',
      createdAt: now,
      emailHash: emailHash,
      status: 'new',
    });
  } catch (error) {
    utils.log('submit-join', 'error', 'record save failed', utils.requestMetadata(event, context, {
      emailHash: emailHash,
      errorName: error && error.name,
      errorMessage: error && error.message,
    }));
    return utils.json(500, { ok: false, error: 'Could not save submission' }, corsHeaders);
  }

  utils.log('submit-join', 'info', 'record saved', utils.requestMetadata(event, context, {
    emailHash: emailHash,
    submissionId: id,
  }));

  var magicLinkResult = { ok: true };
  if (!user) {
    magicLinkResult = await identityMagicLink.sendMagicLink({
      functionName: 'submit-join',
      event: event,
      context: context,
      email: email,
      name: name,
    });
  }

  return utils.json(200, {
    ok: true,
    id: id,
    magicLinkSent: !!magicLinkResult.ok,
    signedIn: !!user,
  }, corsHeaders);
};
