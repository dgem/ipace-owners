/**
 * send-magic-link — Netlify Function
 *
 * Standalone endpoint for requesting a sign-in / confirmation link without
 * submitting the Join form. Join submissions use submit-join, which saves the
 * membership record and calls the same shared Identity helper.
 */

'use strict';

var utils = require('./lib/submission-utils');
var identityMagicLink = require('./lib/identity-magic-link');

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';
  var corsHeaders = utils.corsHeaders(origin);

  utils.log('send-magic-link', 'info', 'request received', utils.requestMetadata(event, context, {
    hasBody: !!event.body,
  }));

  if (origin && !utils.originAllowed(origin)) {
    utils.log('send-magic-link', 'warn', 'request rejected: disallowed origin', utils.requestMetadata(event, context));
    return utils.json(403, { error: 'Forbidden' }, corsHeaders);
  }

  if (event.httpMethod === 'OPTIONS') {
    utils.log('send-magic-link', 'info', 'cors preflight accepted', utils.requestMetadata(event, context));
    return { statusCode: 204, headers: corsHeaders, body: '' };
  }

  if (event.httpMethod !== 'POST') {
    utils.log('send-magic-link', 'warn', 'request rejected: unsupported method', utils.requestMetadata(event, context));
    return utils.json(405, { error: 'Method Not Allowed' }, corsHeaders);
  }

  var body;
  try {
    body = utils.parseBody(event);
  } catch (_) {
    utils.log('send-magic-link', 'warn', 'request rejected: invalid body', utils.requestMetadata(event, context));
    return utils.json(400, { error: 'Invalid request body' }, corsHeaders);
  }

  var email = utils.cleanEmail(body.email);
  var name = utils.cleanString(body.name, 200);

  if (!email || !utils.isEmail(email)) {
    utils.log('send-magic-link', 'warn', 'request rejected: invalid email', utils.requestMetadata(event, context));
    return utils.json(400, { error: 'Valid email address required' }, corsHeaders);
  }

  var result = await identityMagicLink.sendMagicLink({
    functionName: 'send-magic-link',
    event: event,
    context: context,
    email: email,
    name: name,
  });

  return utils.json(200, { ok: !!result.ok }, corsHeaders);
};
