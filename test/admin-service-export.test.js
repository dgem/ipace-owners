const fs = require('node:fs');
const vm = require('node:vm');
const test = require('node:test');
const assert = require('node:assert/strict');

function setup(fetcher, token = () => Promise.resolve('admin-token')) {
  const button = { disabled: false, addEventListener: (_, callback) => { button.click = callback; } };
  const status = {};
  const timers = new Map();
  let timerID = 0;
  const links = [];
  const revoked = [];
  const window = {
    ipaceGetIdentityToken: token,
    ipaceAuthHeaders: (headers) => ({ ...headers, 'X-Ipace-Auth-Trace': 'test-trace' }),
    setTimeout: (callback, delay) => { timers.set(++timerID, { callback, delay }); return timerID; },
    clearTimeout: (id) => timers.delete(id)
  };
  const document = {
    querySelector: (selector) => selector === '[data-service-export]' ? button : status,
    body: { appendChild: (link) => { link.attached = true; links.push(link); } },
    createElement: () => ({
      click() { assert.equal(this.attached, true); this.clicked = true; },
      remove() { this.attached = false; }
    })
  };
  vm.runInNewContext(fs.readFileSync('src/assets/js/admin-service-export.js', 'utf8'), {
    window, document, fetch: fetcher, AbortController,
    URL: { createObjectURL: () => 'blob:test', revokeObjectURL: (url) => revoked.push(url) }
  });
  return { button, status, timers, links, revoked };
}
const flush = () => new Promise((resolve) => setImmediate(resolve));

test('service CSV sends admin token, blocks duplicate clicks, and completes download before cleanup', async () => {
  let calls = 0;
  const ui = setup(async (url, options) => {
    calls++;
    assert.equal(url, '/api/admin/service-export');
    assert.equal(options.headers.Authorization, 'Bearer admin-token');
    assert.equal(options.headers['X-Ipace-Auth-Trace'], 'test-trace');
    return { ok: true, blob: async () => 'csv-data' };
  });
  ui.button.click();
  ui.button.click();
  assert.equal(ui.button.disabled, true);
  assert.match(ui.status.textContent, /Preparing/);
  await flush();
  assert.equal(calls, 1);
  assert.equal(ui.links[0].clicked, true);
  assert.match(ui.links[0].download, /redacted.*csv/);
  assert.equal(ui.revoked.length, 0);
  assert.equal(ui.button.disabled, false);
  assert.match(ui.status.textContent, /downloaded/);
  const cleanup = [...ui.timers.values()].find((timer) => timer.delay === 1000);
  cleanup.callback();
  assert.deepEqual(ui.revoked, ['blob:test']);
});

test('service CSV recovers from access and network failures', async () => {
  for (const fetcher of [async () => ({ ok: false }), async () => { throw new Error('Network unavailable'); }]) {
    const ui = setup(fetcher);
    ui.button.click();
    await flush();
    assert.equal(ui.button.disabled, false);
    assert.equal(ui.links.length, 0);
    assert.match(ui.status.textContent, /Could not export|Network unavailable/);
    assert.equal(ui.timers.size, 0);
  }
});

test('service CSV times out stalled authentication without starting a late download', async () => {
  let resolveToken;
  let calls = 0;
  const ui = setup(async () => { calls++; }, () => new Promise((resolve) => { resolveToken = resolve; }));
  ui.button.click();
  await flush();
  [...ui.timers.values()].find((timer) => timer.delay === 60000).callback();
  await flush();
  assert.equal(ui.button.disabled, false);
  assert.match(ui.status.textContent, /too long/);
  resolveToken('late-token');
  await flush();
  assert.equal(calls, 0);
  assert.equal(ui.links.length, 0);
});
