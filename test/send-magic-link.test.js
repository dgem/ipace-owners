'use strict';

var assert = require('node:assert/strict');
var test = require('node:test');
var sendMagicLink = require('../netlify/functions/send-magic-link');

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

test('send-magic-link calls signup for new users', async function (t) {
  var originalFetch = global.fetch;
  var originalBaseUrl = process.env.NETLIFY_IDENTITY_BASE_URL;
  var calls = [];

  t.after(function () {
    global.fetch = originalFetch;
    if (originalBaseUrl === undefined) {
      delete process.env.NETLIFY_IDENTITY_BASE_URL;
    } else {
      process.env.NETLIFY_IDENTITY_BASE_URL = originalBaseUrl;
    }
  });

  process.env.NETLIFY_IDENTITY_BASE_URL = 'https://example.netlify.app/.netlify/identity';
  global.fetch = async function (url, options) {
    calls.push({ url: url, options: options });
    return { ok: true, status: 200 };
  };

  var res = await sendMagicLink.handler(event({ email: 'USER@example.com', name: 'User' }), {});
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.ok, true);
  assert.equal(calls.length, 1);
  assert.equal(calls[0].url, 'https://example.netlify.app/.netlify/identity/signup');
  assert.equal(JSON.parse(calls[0].options.body).email, 'user@example.com');
});

test('send-magic-link calls passwordless magiclink when signup reports an existing account', async function (t) {
  var originalFetch = global.fetch;
  var originalBaseUrl = process.env.NETLIFY_IDENTITY_BASE_URL;
  var calls = [];

  t.after(function () {
    global.fetch = originalFetch;
    if (originalBaseUrl === undefined) {
      delete process.env.NETLIFY_IDENTITY_BASE_URL;
    } else {
      process.env.NETLIFY_IDENTITY_BASE_URL = originalBaseUrl;
    }
  });

  process.env.NETLIFY_IDENTITY_BASE_URL = 'https://example.netlify.app/.netlify/identity';
  global.fetch = async function (url, options) {
    calls.push({ url: url, options: options });
    if (url.endsWith('/signup')) return { ok: false, status: 422 };
    return { ok: true, status: 200 };
  };

  var res = await sendMagicLink.handler(event({ email: 'member@example.com' }), {});
  var body = JSON.parse(res.body);

  assert.equal(res.statusCode, 200);
  assert.equal(body.ok, true);
  assert.equal(calls.length, 2);
  assert.equal(calls[0].url, 'https://example.netlify.app/.netlify/identity/signup');
  assert.equal(calls[1].url, 'https://example.netlify.app/.netlify/identity/magiclink');
  assert.deepEqual(JSON.parse(calls[1].options.body), { email: 'member@example.com' });
});

test('send-magic-link rejects disallowed origins before calling Identity', async function (t) {
  var originalFetch = global.fetch;
  var calls = 0;

  t.after(function () {
    global.fetch = originalFetch;
  });

  global.fetch = async function () {
    calls += 1;
    return { ok: true, status: 200 };
  };

  var res = await sendMagicLink.handler({
    httpMethod: 'POST',
    headers: { origin: 'https://evil.example', 'content-type': 'application/json' },
    body: JSON.stringify({ email: 'user@example.com' }),
  }, {});

  assert.equal(res.statusCode, 403);
  assert.equal(calls, 0);
});
