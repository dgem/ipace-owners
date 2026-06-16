'use strict';

var crypto = require('crypto');
var utils = require('./submission-utils');

function isLocalOrigin(origin) {
  return /^http:\/\/localhost(?::\d+)?$/.test(origin || '');
}

function resolveIdentityBase(event) {
  if (process.env.NETLIFY_IDENTITY_BASE_URL) {
    return process.env.NETLIFY_IDENTITY_BASE_URL.replace(/\/$/, '');
  }

  var origin = event && event.headers && (event.headers.origin || event.headers.Origin) || '';
  if (origin && utils.originAllowed(origin) && !isLocalOrigin(origin)) {
    return origin.replace(/\/$/, '') + '/.netlify/identity';
  }

  var siteUrl = (process.env.URL || process.env.DEPLOY_PRIME_URL || '').replace(/\/$/, '');
  if (!siteUrl || isLocalOrigin(siteUrl)) {
    return '';
  }

  return siteUrl + '/.netlify/identity';
}

async function sendMagicLink(options) {
  var functionName = options.functionName;
  var event = options.event;
  var context = options.context;
  var email = options.email;
  var name = options.name || '';
  var emailHash = utils.emailFingerprint(email);
  var identityBase = resolveIdentityBase(event);

  if (!identityBase) {
    utils.log(functionName, 'error', 'identity base url missing', utils.requestMetadata(event, context, {
      emailHash: emailHash,
      hasNetlifyIdentityBaseUrl: !!process.env.NETLIFY_IDENTITY_BASE_URL,
      hasUrl: !!process.env.URL,
      hasDeployPrimeUrl: !!process.env.DEPLOY_PRIME_URL,
    }));
    return { ok: false };
  }

  utils.log(functionName, 'info', 'identity request starting', utils.requestMetadata(event, context, {
    emailHash: emailHash,
    identityBaseHost: new URL(identityBase).host,
    hasName: !!name,
  }));

  var randomPassword = crypto.randomBytes(18).toString('hex') + 'Aa1!';

  try {
    var signupRes = await fetch(identityBase + '/signup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, password: randomPassword, data: { full_name: name } }),
    });

    utils.log(functionName, 'info', 'identity signup response received', utils.requestMetadata(event, context, {
      emailHash: emailHash,
      status: signupRes.status,
      ok: signupRes.ok,
    }));

    if (signupRes.status === 422) {
      var magicLinkRes = await fetch(identityBase + '/magiclink', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email }),
      });

      utils.log(functionName, 'info', 'identity magiclink response received', utils.requestMetadata(event, context, {
        emailHash: emailHash,
        status: magicLinkRes.status,
        ok: magicLinkRes.ok,
      }));

      return { ok: magicLinkRes.ok };
    }

    if (!signupRes.ok) {
      utils.log(functionName, 'warn', 'identity signup failed', utils.requestMetadata(event, context, {
        emailHash: emailHash,
        status: signupRes.status,
      }));
      return { ok: false };
    }
  } catch (error) {
    utils.log(functionName, 'error', 'identity request threw', utils.requestMetadata(event, context, {
      emailHash: emailHash,
      errorName: error && error.name,
      errorMessage: error && error.message,
    }));
    return { ok: false };
  }

  utils.log(functionName, 'info', 'magic link flow completed', utils.requestMetadata(event, context, {
    emailHash: emailHash,
  }));
  return { ok: true };
}

module.exports = {
  resolveIdentityBase: resolveIdentityBase,
  sendMagicLink: sendMagicLink,
};
