/**
 * send-magic-link — Netlify Function
 *
 * Accepts: POST { email: string, name?: string }
 *
 * Sends a sign-in / confirmation link to the supplied email address by calling
 * the Netlify Identity API server-side. For new addresses it calls /signup; for
 * existing ones it calls /recover (password-reset link that also signs the user
 * in when clicked). For valid same-origin requests, the function returns HTTP
 * 200 with an `ok` flag so an observer cannot distinguish registered from
 * unregistered email addresses (account enumeration prevention).
 *
 * Environment requirements (set automatically by Netlify):
 *   URL — the primary URL of the site, e.g. https://ipace-owners.netlify.app
 *         Used to build the /.netlify/identity endpoint.
 *
 * Local development:
 *   Set NETLIFY_IDENTITY_BASE_URL to the deployed site's Identity endpoint,
 *   e.g. https://ipace-owners.netlify.app/.netlify/identity. Netlify Dev
 *   serves Functions locally, but it does not provide a local Identity API.
 */

'use strict';

var crypto = require('crypto');

var ALLOWED_ORIGINS = [
  'https://ipace-owners.org',
  'https://ipace-owners.netlify.app',
];

function log(level, message, data) {
  var payload = Object.assign({
    level: level,
    message: message,
    function: 'send-magic-link',
  }, data || {});

  if (level === 'error') {
    console.error(JSON.stringify(payload));
  } else if (level === 'warn') {
    console.warn(JSON.stringify(payload));
  } else {
    console.log(JSON.stringify(payload));
  }
}

function emailFingerprint(email) {
  return crypto
    .createHash('sha256')
    .update(email)
    .digest('hex')
    .slice(0, 16);
}

function requestMetadata(event, context, extra) {
  return Object.assign({
    requestId: context && context.awsRequestId,
    method: event.httpMethod,
    origin: (event.headers && (event.headers.origin || event.headers.Origin)) || 'none',
  }, extra || {});
}

function resolveIdentityBase() {
  if (process.env.NETLIFY_IDENTITY_BASE_URL) {
    return process.env.NETLIFY_IDENTITY_BASE_URL.replace(/\/$/, '');
  }

  var siteUrl = (process.env.URL || process.env.DEPLOY_PRIME_URL || '').replace(/\/$/, '');
  if (!siteUrl || /^http:\/\/localhost(?::\d+)?$/.test(siteUrl)) {
    return '';
  }

  return siteUrl + '/.netlify/identity';
}

/**
 * Returns true if the request origin is the same site (including deploy previews
 * and local dev servers) or an explicitly allowed production origin.
 */
function originAllowed(origin) {
  if (!origin) return false;
  if (ALLOWED_ORIGINS.indexOf(origin) !== -1) return true;
  // Allow Netlify deploy-preview and branch-deploy URLs for this site.
  if (/^https:\/\/[a-z0-9-]+--ipace-owners\.netlify\.app$/.test(origin)) return true;
  // Allow local development servers (any localhost port).
  if (/^http:\/\/localhost(:\d+)?$/.test(origin)) return true;
  return false;
}

exports.handler = async function (event, context) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';

  log('info', 'request received', requestMetadata(event, context, {
    hasBody: !!event.body,
  }));

  // If an Origin header is present but is not on the allowlist, reject the
  // request outright. This prevents cross-site requests from triggering the
  // side effect (sending emails) even via no-cors fetch, which would bypass
  // browser-enforced CORS read restrictions but still hit the Function.
  // Requests with no Origin header (server-to-server, curl, etc.) are allowed
  // through — they are not subject to browser CORS and cannot be spoofed by
  // arbitrary web pages.
  if (origin && !originAllowed(origin)) {
    log('warn', 'request rejected: disallowed origin', requestMetadata(event, context));
    return {
      statusCode: 403,
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ error: 'Forbidden' }),
    };
  }

  // CORS headers — only reflect Allow-Origin for allowlisted origins.
  var corsHeaders = {
    'Access-Control-Allow-Methods': 'POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type',
  };
  if (origin) {
    corsHeaders['Access-Control-Allow-Origin'] = origin;
  }

  // Handle CORS preflight.
  if (event.httpMethod === 'OPTIONS') {
    log('info', 'cors preflight accepted', requestMetadata(event, context));
    return { statusCode: 204, headers: corsHeaders, body: '' };
  }

  if (event.httpMethod !== 'POST') {
    log('warn', 'request rejected: unsupported method', requestMetadata(event, context));
    return { statusCode: 405, headers: corsHeaders, body: 'Method Not Allowed' };
  }

  // Parse body.
  var email, name;
  try {
    var body = JSON.parse(event.body || '{}');
    email = String(body.email || '').trim().toLowerCase();
    name  = String(body.name  || '').trim().slice(0, 200);
  } catch (_) {
    log('warn', 'request rejected: invalid json', requestMetadata(event, context));
    return {
      statusCode: 400,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ error: 'Invalid JSON body' }),
    };
  }

  // Basic email validation — reject obviously invalid values early.
  if (!email || !(/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email))) {
    log('warn', 'request rejected: invalid email', requestMetadata(event, context));
    return {
      statusCode: 400,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ error: 'Valid email address required' }),
    };
  }

  // Build the Identity API base URL from the deployed site's URL. Netlify Dev
  // does not expose /.netlify/identity locally, so localhost is not a valid
  // Identity base URL.
  var identityBase = resolveIdentityBase();
  var emailHash = emailFingerprint(email);

  if (!identityBase) {
    log('error', 'identity base url missing', requestMetadata(event, context, {
      emailHash: emailHash,
      hasNetlifyIdentityBaseUrl: !!process.env.NETLIFY_IDENTITY_BASE_URL,
      hasUrl: !!process.env.URL,
      hasDeployPrimeUrl: !!process.env.DEPLOY_PRIME_URL,
    }));
    return {
      statusCode: 200,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ ok: false }),
    };
  }

  log('info', 'identity request starting', requestMetadata(event, context, {
    emailHash: emailHash,
    identityBaseHost: new URL(identityBase).host,
    hasName: !!name,
  }));

  // Generate a random password. The user never sees or uses this; they
  // authenticate exclusively via the emailed confirmation/recovery link.
  var randomPassword = crypto.randomBytes(18).toString('hex') + 'Aa1!';

  try {
    // Attempt signup for new users. A confirmation email (magic sign-in link)
    // is sent automatically when autoconfirm is disabled in Netlify Identity.
    var signupRes = await fetch(identityBase + '/signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, password: randomPassword, data: { full_name: name } }),
    });

    log('info', 'identity signup response received', requestMetadata(event, context, {
      emailHash: emailHash,
      status: signupRes.status,
      ok: signupRes.ok,
    }));

    if (signupRes.status === 422) {
      // Email already registered — send a recovery / sign-in link instead.
      var recoverRes = await fetch(identityBase + '/recover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email }),
      });

      log('info', 'identity recover response received', requestMetadata(event, context, {
        emailHash: emailHash,
        status: recoverRes.status,
        ok: recoverRes.ok,
      }));

      if (!recoverRes.ok) {
        return {
          statusCode: 200,
          headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
          body: JSON.stringify({ ok: false }),
        };
      }
    } else if (!signupRes.ok) {
      // Non-422 Identity error (e.g. misconfiguration, outage). This failure
      // is not related to whether the email is registered, so we can safely
      // signal it to the UI without enabling account enumeration.
      log('warn', 'identity signup failed', requestMetadata(event, context, {
        emailHash: emailHash,
        status: signupRes.status,
      }));
      return {
        statusCode: 200,
        headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
        body: JSON.stringify({ ok: false }),
      };
    }
  } catch (error) {
    // Network or unexpected error — signal failure to the UI.
    log('error', 'identity request threw', requestMetadata(event, context, {
      emailHash: emailHash,
      errorName: error && error.name,
      errorMessage: error && error.message,
    }));
    return {
      statusCode: 200,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ ok: false }),
    };
  }

  // Success: email dispatched (or recovery sent for existing accounts).
  log('info', 'magic link flow completed', requestMetadata(event, context, {
    emailHash: emailHash,
  }));
  return {
    statusCode: 200,
    headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
    body: JSON.stringify({ ok: true }),
  };
};
