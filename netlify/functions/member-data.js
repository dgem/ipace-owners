'use strict';

var utils = require('./lib/submission-utils');

/**
 * member-data — Netlify Function
 *
 * Returns the authenticated user's own join and vehicle-basics records.
 * Authorisation is enforced server-side via context.clientContext.user.sub.
 */

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';
  var corsHeaders = utils.corsHeaders(origin);

  utils.log('member-data', 'info', 'request received', utils.requestMetadata(event, context, {
    hasBody: !!event.body,
   }));

  if (origin && !utils.originAllowed(origin)) {
    utils.log('member-data', 'warn', 'request rejected: disallowed origin', utils.requestMetadata(event, context));
    return utils.json(403, { error: 'Forbidden' }, corsHeaders);
   }

  if (event.httpMethod === 'OPTIONS') {
    return { statusCode: 204, headers: corsHeaders, body: '' };
   }

  if (event.httpMethod !== 'GET') {
    return utils.json(405, { error: 'Method Not Allowed' }, corsHeaders);
   }

  var user = utils.requireUser(context);
  if (!user || !user.sub) {
    utils.log('member-data', 'warn', 'request rejected: not authenticated', utils.requestMetadata(event, context));
    return utils.json(401, { error: 'Sign in required' }, corsHeaders);
   }

  var result = {
    identityUserId: user.sub,
    email: user.email || null,
    joinRecords: [],
    vehicleRecords: [],
   };

  // ── Fetch join records (by identityUserId) ────────────────────────────────────
  try {
    var joinRecords = await utils.listJsonRecords(event, 'join/');
    for (var i = 0; i < joinRecords.length; i++) {
      var record = joinRecords[i];
      if (record && record.identityUserId === user.sub) {
        result.joinRecords.push({
          id: record.id,          identityUserId: record.identityUserId,          createdAt: record.createdAt,
          updatedAt: record.updatedAt,
          contact: record.contact,
          membership: record.membership,
          consents: record.consents,
         });
       }
     }
   } catch (joinErr) {
    utils.log('member-data', 'error', 'failed to list join records', utils.requestMetadata(event, context, {
      errorName: joinErr && joinErr.name,
      errorMessage: joinErr && joinErr.message,
     }));
   }

  // ── Fetch vehicle-basics records (scoped by user.sub in path) ─────────────────
  try {
    var vehiclePrefix = 'vehicle-basics/' + user.sub + '/';
    var vehicleRecords = await utils.listJsonRecords(event, vehiclePrefix);
    for (var j = 0; j < vehicleRecords.length; j++) {
      var vRecord = vehicleRecords[j];
      if (vRecord && vRecord.identityUserId === user.sub) {
        result.vehicleRecords.push({
          id: vRecord.id,
          identityUserId: vRecord.identityUserId,
          createdAt: vRecord.createdAt,
          updatedAt: vRecord.updatedAt,
          vehicle: vRecord.vehicle,
          battery: vRecord.battery,
         });
       }
     }
   } catch (vehicleErr) {
    utils.log('member-data', 'error', 'failed to list vehicle records', utils.requestMetadata(event, context, {
      errorName: vehicleErr && vehicleErr.name,
      errorMessage: vehicleErr && vehicleErr.message,
     }));
   }

  return utils.json(200, result, corsHeaders);
};
