'use strict';
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { magicLinkFromMessage } = require('../scripts/auth-e2e-link.cjs');
const origin = 'https://preview.example';
function link(continuation) {
  const url = new URL('/auth/action', origin);
  url.search = new URLSearchParams({ mode: 'signIn', oobCode: 'test-code', apiKey: 'test-key', continueUrl: continuation }).toString();
  return url.href;
}
test('accepts a sign-in link with a valid support trace', () => {
  for (const suffix of ['', '?authTrace=IP-ABCD-2345']) {
    const candidate = link(origin + '/member/account/' + suffix);
    assert.equal(magicLinkFromMessage({ text: candidate }, origin), candidate);
  }
});
test('rejects unsafe continuation URLs and traces', () => {
  for (const continuation of ['https://other.example/member/account/', origin + '/member/account/?next=https://other.example', origin + '/member/account/?authTrace=bad', origin + '/member/account/?authTrace=IP-ABCD-2345&authTrace=IP-ABCD-2345']) {
    assert.equal(magicLinkFromMessage({ text: link(continuation) }, origin), undefined);
  }
});
