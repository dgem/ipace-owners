'use strict';

var utils = require('./lib/submission-utils');

/**
 * admin-data — Netlify Function
 *
 * Returns all join and vehicle-basics records for administrators.
 * Authorisation is enforced server-side: the user must have the 'admin' role
 * in app_metadata.roles (set via Netlify Identity UI).
 */

function isAdmin(user) {
  if (!user) return false;
  var roles = (user.app_metadata && user.app_metadata.roles) || [];
  return roles.indexOf('admin') !== -1;
}

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';
  var corsHeaders = utils.corsHeaders(origin);

  utils.log('admin-data', 'info', 'request received', utils.requestMetadata(event, context, {
    hasBody: !!event.body,
    }));

  if (origin && !utils.originAllowed(origin)) {
    utils.log('admin-data', 'warn', 'request rejected: disallowed origin', utils.requestMetadata(event, context));
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
    utils.log('admin-data', 'warn', 'request rejected: not authenticated', utils.requestMetadata(event, context));
    return utils.json(401, { error: 'Sign in required' }, corsHeaders);
    }

  if (!isAdmin(user)) {
    utils.log('admin-data', 'warn', 'request rejected: not admin', utils.requestMetadata(event, context, {
      userHash: utils.emailFingerprint(user.sub),
      }));
    return utils.json(403, { error: 'Admin access required' }, corsHeaders);
    }

  var result = {
    identityUserId: user.sub,
    email: user.email || null,
    joinRecords: [],
    vehicleRecords: [],
    };

   // ── Fetch all join records ───────────────────────────────────────────────────
  try {
    var joinStore = utils.getStore(event);
    var joinKeys = await joinStore.list('join/');
    for (var i = 0; i < joinKeys.length; i++) {
      var record = await joinKeys[i].getJSON();
      if (record) {
        result.joinRecords.push({
          id: record.id,
          createdAt: record.createdAt,
          updatedAt: record.updatedAt,
          identityUserId: record.identityUserId,
          contact: record.contact,
          membership: record.membership,
          consents: record.consents,
          review: record.review,
          });
        }
      }
    } catch (joinErr) {
    utils.log('admin-data', 'error', 'failed to list join records', utils.requestMetadata(event, context, {
      errorName: joinErr && joinErr.name,
      errorMessage: joinErr && joinErr.message,
      }));
    }

   // ── Fetch all vehicle-basics records ────────────────────────────────────────
  try {
    var vehicleStore = utils.getStore(event);
    var vehicleKeys = await vehicleStore.list('vehicle-basics/');
    for (var j = 0; j < vehicleKeys.length; j++) {
      var vRecord = await vehicleKeys[j].getJSON();
      if (vRecord) {
        result.vehicleRecords.push({
          id: vRecord.id,
          createdAt: vRecord.createdAt,
          updatedAt: vRecord.updatedAt,
          identityUserId: vRecord.identityUserId,
          userEmailHash: vRecord.userEmailHash,
          vehicle: vRecord.vehicle,
          battery: vRecord.battery,
          review: vRecord.review,
          });
        }
      }
    } catch (vehicleErr) {
    utils.log('admin-data', 'error', 'failed to list vehicle records', utils.requestMetadata(event, context, {
      errorName: vehicleErr && vehicleErr.name,
      errorMessage: vehicleErr && vehicleErr.message,
      }));
    }

  return utils.json(200, result, corsHeaders);
};
