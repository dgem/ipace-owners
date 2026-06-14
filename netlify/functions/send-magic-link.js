/**
 * send-magic-link — Netlify Function
 *
 * Accepts: POST { email: string, name?: string }
 *
 * Sends a sign-in / confirmation link to the supplied email address by calling
 * the Netlify Identity API server-side. For new addresses it calls /signup; for
 * existing ones it calls /recover (password-reset link that also signs the user
 * in when clicked). The function ALWAYS returns HTTP 200 to the browser,
 * regardless of what the Identity API returns, so that an observer cannot
 * distinguish registered from unregistered email addresses (account enumeration
 * prevention).
 *
 * Environment requirements (set automatically by Netlify):
 *   URL — the primary URL of the site, e.g. https://ipace-owners.netlify.app
 *         Used to build the /.netlify/identity endpoint.
 */

'use strict';

var crypto = require('crypto');

var ALLOWED_ORIGINS = [
  'https://ipace-owners.org',
  'https://ipace-owners.netlify.app',
];

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

exports.handler = async function (event) {
  var origin = (event.headers && (event.headers.origin || event.headers.Origin)) || '';

  // If an Origin header is present but is not on the allowlist, reject the
  // request outright. This prevents cross-site requests from triggering the
  // side effect (sending emails) even via no-cors fetch, which would bypass
  // browser-enforced CORS read restrictions but still hit the Function.
  // Requests with no Origin header (server-to-server, curl, etc.) are allowed
  // through — they are not subject to browser CORS and cannot be spoofed by
  // arbitrary web pages.
  if (origin && !originAllowed(origin)) {
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
    return { statusCode: 204, headers: corsHeaders, body: '' };
  }

  if (event.httpMethod !== 'POST') {
    return { statusCode: 405, headers: corsHeaders, body: 'Method Not Allowed' };
  }

  // Parse body.
  var email, name;
  try {
    var body = JSON.parse(event.body || '{}');
    email = String(body.email || '').trim().toLowerCase();
    name  = String(body.name  || '').trim().slice(0, 200);
  } catch (_) {
    return {
      statusCode: 400,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ error: 'Invalid JSON body' }),
    };
  }

  // Basic email validation — reject obviously invalid values early.
  if (!email || !(/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email))) {
    return {
      statusCode: 400,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ error: 'Valid email address required' }),
    };
  }

  // Build the Identity API base URL from the site's primary URL.
  // process.env.URL is set automatically by Netlify to the site's primary URL.
  var siteUrl = (process.env.URL || '').replace(/\/$/, '');
  if (!siteUrl) {
    // Fallback for local dev — will fail to reach Identity but won't throw.
    siteUrl = 'http://localhost:8888';
  }
  var identityBase = siteUrl + '/.netlify/identity';

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

    if (signupRes.status === 422) {
      // Email already registered — send a recovery / sign-in link instead.
      await fetch(identityBase + '/recover', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email }),
      });
    } else if (!signupRes.ok) {
      // Non-422 Identity error (e.g. misconfiguration, outage). This failure
      // is not related to whether the email is registered, so we can safely
      // signal it to the UI without enabling account enumeration.
      return {
        statusCode: 200,
        headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
        body: JSON.stringify({ ok: false }),
      };
    }
  } catch (_) {
    // Network or unexpected error — signal failure to the UI.
    return {
      statusCode: 200,
      headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
      body: JSON.stringify({ ok: false }),
    };
  }

  // Success: email dispatched (or recovery sent for existing accounts).
  return {
    statusCode: 200,
    headers: Object.assign({ 'Content-Type': 'application/json' }, corsHeaders),
    body: JSON.stringify({ ok: true }),
  };
};
