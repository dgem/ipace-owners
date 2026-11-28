const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

test('opted-in member without a Join UID is shown and approved by email', async function () {
  const handlers = {};
  const status = { textContent: '' };
  const results = { innerHTML: '', addEventListener: (_, fn) => { handlers.approve = fn; } };
  const load = { addEventListener: (_, fn) => { handlers.load = fn; } };
  const root = { querySelector: (selector) => ({ '[data-feature-testers-status]': status, '[data-feature-testers-results]': results, '[data-load-feature-testers]': load }[selector]) };
  const requests = [];
  const document = { querySelector: () => root, createElement: () => ({ set textContent(value) { this.innerHTML = String(value); }, innerHTML: '' }) };
  const fetch = async (url, options) => { requests.push({ url, options }); return { ok: true, json: async () => url.includes('/members?') ? { results: [{ name: 'Member', email: 'member@example.org', betaTester: true, identityUserId: '' }] } : { message: 'Approved' } }; };
  vm.runInNewContext(fs.readFileSync('src/assets/js/admin-feature-testers.js', 'utf8'), { document, fetch, window: { ipaceGetIdentityToken: async () => 'token' } });
  handlers.load();
  await new Promise(setImmediate);
  assert.match(results.innerHTML, /member@example\.org/);
  assert.match(results.innerHTML, /data-approve-feature-tester/);
  const button = { dataset: { uid: '', email: 'member@example.org' }, disabled: false };
  handlers.approve({ target: { closest: () => button } });
  await new Promise(setImmediate);
  assert.deepEqual(JSON.parse(requests[1].options.body), { uid: '', email: 'member@example.org', approved: true });
  assert.equal(status.textContent, 'Approved');
});
