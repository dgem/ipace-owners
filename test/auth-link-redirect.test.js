const assert = require('node:assert/strict');
const test = require('node:test');
const config = require('../firebase.json');

test('branded auth routes use fixed same-site temporary redirects', () => {
  for (const path of ['/auth/action', '/auth/action/']) {
    const rule = config.hosting.redirects.find((entry) => entry.source === path);
    assert.deepEqual(rule, { source: path, destination: '/__/auth/action', type: 302 });
  }
  const headers = Object.fromEntries(config.hosting.headers.filter((entry) => ['**', '/auth/**'].includes(entry.source)).flatMap((entry) => entry.headers.map(({ key, value }) => [key, value])));
  assert.match(headers['Cache-Control'], /no-store/);
  assert.equal(headers['Referrer-Policy'], 'no-referrer');
});

test('auth redirect smoke check preserves encoded values without following sign-in links', async () => {
  const { checkAuthLinkRedirect } = await import('../scripts/auth-link-redirect-smoke.mjs');
  const paths = [];
  await checkAuthLinkRedirect('https://example.web.app', async (input, options) => {
    const request = new URL(input);
    paths.push(request.pathname);
    assert.equal(options.redirect, 'manual');
    assert.equal(request.searchParams.get('oobCode'), 'smoke+dummy/code');
    assert.equal(request.searchParams.get('continueUrl'), 'https://example.web.app/member/account/?authTrace=smoke&next=survey');
    return new Response(null, { status: 302, headers: { Location: '/__/auth/action' + request.search } });
  });
  assert.deepEqual(paths, ['/auth/action', '/auth/action/']);
});

test('auth redirect smoke check rejects lost parameters, wrong hosts and permanent redirects', async () => {
  const { checkAuthLinkRedirect } = await import('../scripts/auth-link-redirect-smoke.mjs');
  for (const status of [301, 200]) {
    await assert.rejects(checkAuthLinkRedirect('https://example.web.app', async (input) => new Response(null, {
      status, headers: { Location: '/__/auth/action' + new URL(input).search }
    })));
  }
  for (const location of ['/__/auth/action', 'https://foreign.example/__/auth/action']) {
    await assert.rejects(checkAuthLinkRedirect('https://example.web.app', async () => new Response(null, {
      status: 302, headers: { Location: location }
    })));
  }
});
