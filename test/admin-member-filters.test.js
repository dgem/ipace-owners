const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');

test('member filters work without a search term and tester approval is explicit', async function () {
  const handlers = {};
  const query = { value: '' }, preference = { value: 'tester' }, country = { value: 'unset' };
  const status = { textContent: '' }, results = { innerHTML: '', addEventListener: (_, fn) => { handlers.approve = fn; } };
  const prev = { hidden: true, addEventListener: (_, fn) => { handlers.prev = fn; } }, next = { hidden: true, addEventListener: (_, fn) => { handlers.next = fn; } };
  const form = { addEventListener: (_, fn) => { handlers.submit = fn; } };
  const nodes = { '[data-results]': results, '[data-status]': status, '[data-previous]': prev, '[data-next]': next, form, '#admin-member-query': query, '#admin-member-preference': preference, '#admin-member-country': country };
  const root = { querySelector: (selector) => nodes[selector] };
  const requests = [];
  const fetch = async (url, options) => { requests.push({ url, options }); return { ok: true, json: async () => url.includes('/members?') ? { total: 51, hasMore: true, results: [{ name: 'Member', email: 'member@example.org', betaTester: true, previewApproved: false }] } : { message: 'Approved' } }; };
  const document = { querySelector: () => root, createElement: () => ({ set textContent(value) { this.innerHTML = String(value); }, innerHTML: '' }) };
  vm.runInNewContext(fs.readFileSync('src/assets/js/admin-member-search.js', 'utf8'), { document, fetch, URLSearchParams, window: { ipaceGetIdentityToken: async () => 'token' } });
  handlers.submit({ preventDefault() {} }); await new Promise(setImmediate);
  assert.match(requests[0].url, /preference=tester/); assert.match(requests[0].url, /country=unset/); assert.match(results.innerHTML, /Approve feature preview/); assert.equal(next.hidden, false);
  const button = { dataset: { uid: '', email: 'member@example.org', approved: 'true' }, disabled: false };
  handlers.approve({ target: { closest: () => button } }); await new Promise(setImmediate);
  assert.deepEqual(JSON.parse(requests[1].options.body), { uid: '', email: 'member@example.org', approved: true });
});
