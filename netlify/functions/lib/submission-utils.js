'use strict';

var crypto = require('crypto');
var blobs = require('@netlify/blobs');

var ALLOWED_ORIGINS = [
  'https://ipace-owners.org',
  'https://ipace-owners.netlify.app',
];

function log(functionName, level, message, data) {
  var payload = Object.assign({
    level: level,
    message: message,
    function: functionName,
  }, data || {});

  if (level === 'error') {
    console.error(JSON.stringify(payload));
  } else if (level === 'warn') {
    console.warn(JSON.stringify(payload));
  } else {
    console.log(JSON.stringify(payload));
  }
}

function originAllowed(origin) {
  if (!origin) return false;
  if (ALLOWED_ORIGINS.indexOf(origin) !== -1) return true;
  if (/^https:\/\/[a-z0-9-]+--ipace-owners\.netlify\.app$/.test(origin)) return true;
  if (/^http:\/\/localhost(:\d+)?$/.test(origin)) return true;
  return false;
}

function requestMetadata(event, context, extra) {
  return Object.assign({
    requestId: context && context.awsRequestId,
    method: event.httpMethod,
    origin: (event.headers && (event.headers.origin || event.headers.Origin)) || 'none',
  }, extra || {});
}

function corsHeaders(origin) {
  var headers = {
    'Access-Control-Allow-Methods': 'POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization',
  };
  if (origin) headers['Access-Control-Allow-Origin'] = origin;
  return headers;
}

function json(statusCode, body, headers) {
  return {
    statusCode: statusCode,
    headers: Object.assign({ 'Content-Type': 'application/json' }, headers || {}),
    body: JSON.stringify(body),
  };
}

function parseBody(event) {
  var raw = event.body || '';
  var contentType = String((event.headers && (event.headers['content-type'] || event.headers['Content-Type'])) || '').toLowerCase();

  if (contentType.indexOf('application/x-www-form-urlencoded') !== -1) {
    var params = new URLSearchParams(raw);
    var parsed = {};
    params.forEach(function (value, key) {
      if (Object.prototype.hasOwnProperty.call(parsed, key)) {
        if (!Array.isArray(parsed[key])) parsed[key] = [parsed[key]];
        parsed[key].push(value);
      } else {
        parsed[key] = value;
      }
    });
    return parsed;
  }

  if (!raw) return {};
  return JSON.parse(raw);
}

function cleanString(value, maxLength) {
  return String(value || '').trim().slice(0, maxLength || 500);
}

function cleanEmail(value) {
  return cleanString(value, 254).toLowerCase();
}

function isEmail(value) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

function cleanEnum(value, allowed) {
  var cleaned = cleanString(value, 100);
  return allowed.indexOf(cleaned) !== -1 ? cleaned : '';
}

function cleanDate(value) {
  var cleaned = cleanString(value, 20);
  return /^\d{4}-\d{2}-\d{2}$/.test(cleaned) ? cleaned : '';
}

function cleanInteger(value, min, max) {
  if (value === undefined || value === null || value === '') return null;
  var parsed = Number(value);
  if (!Number.isInteger(parsed)) return null;
  if (typeof min === 'number' && parsed < min) return null;
  if (typeof max === 'number' && parsed > max) return null;
  return parsed;
}

function cleanDecimal(value, min, max) {
  if (value === undefined || value === null || value === '') return null;
  var parsed = Number(value);
  if (!Number.isFinite(parsed)) return null;
  if (typeof min === 'number' && parsed < min) return null;
  if (typeof max === 'number' && parsed > max) return null;
  return Math.round(parsed * 10) / 10;
}

function asArray(value) {
  if (Array.isArray(value)) return value;
  if (value === undefined || value === null || value === '') return [];
  return [value];
}

function filterEnums(value, allowed) {
  return asArray(value).map(function (item) {
    return cleanEnum(item, allowed);
  }).filter(Boolean);
}

function emailFingerprint(email) {
  return crypto
    .createHash('sha256')
    .update(email)
    .digest('hex')
    .slice(0, 16);
}

function hmac(value, secret) {
  return crypto
    .createHmac('sha256', secret)
    .update(value)
    .digest('hex');
}

function requireUser(context) {
  return context && context.clientContext && context.clientContext.user
    ? context.clientContext.user
    : null;
}

function getStore(event) {
  blobs.connectLambda(event);
  return blobs.getStore('owner-submissions');
}

async function saveRecord(event, key, record, metadata) {
  var store = getStore(event);
  await store.setJSON(key, record, { metadata: metadata || {} });
}

function submissionId(prefix) {
  return prefix + '_' + crypto.randomUUID();
}

module.exports = {
  cleanDate: cleanDate,
  cleanDecimal: cleanDecimal,
  cleanEmail: cleanEmail,
  cleanEnum: cleanEnum,
  cleanInteger: cleanInteger,
  cleanString: cleanString,
  corsHeaders: corsHeaders,
  emailFingerprint: emailFingerprint,
  filterEnums: filterEnums,
  hmac: hmac,
  isEmail: isEmail,
  json: json,
  log: log,
  originAllowed: originAllowed,
  parseBody: parseBody,
  requestMetadata: requestMetadata,
  requireUser: requireUser,
  saveRecord: saveRecord,
  submissionId: submissionId,
};
