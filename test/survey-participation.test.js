const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const assert = require('node:assert/strict');

const script = fs.readFileSync('src/assets/js/survey-participation.js', 'utf8');

async function render(response) {
  const value = { textContent: '540' };
  const date = { textContent: '19 September 2026' };
  vm.runInNewContext(script, {
    document: { querySelector: () => ({ querySelector: (selector) => selector.includes('__value') ? value : date }) },
    fetch: () => Promise.resolve(response)
  });
  await new Promise((resolve) => setImmediate(resolve));
  return [value.textContent, date.textContent];
}

test('participation counter accepts live zero and formatted counts', async () => {
  assert.deepEqual(await render({ ok: true, json: async () => ({ responses: 0 }) }), ['0', 'Latest response total']);
  assert.deepEqual(await render({ ok: true, json: async () => ({ responses: 1540 }) }), ['1,540', 'Latest response total']);
});

test('participation counter preserves its dated fallback on errors and invalid counts', async () => {
  for (const responses of [-1, 1.5, '600', null, Number.MAX_SAFE_INTEGER + 1]) {
    assert.deepEqual(await render({ ok: true, json: async () => ({ responses }) }), ['540', '19 September 2026']);
  }
  assert.deepEqual(await render({ ok: false }), ['540', '19 September 2026']);
});
