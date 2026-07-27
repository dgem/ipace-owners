const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');

test('member account offers authenticated CSV and Excel exports', () => {
  const account = fs.readFileSync(path.join(root, 'src/member/account.njk'), 'utf8');
  const script = fs.readFileSync(path.join(root, 'src/assets/js/member-export.js'), 'utf8');
  const main = fs.readFileSync(path.join(root, 'functions/firebase-go/main.go'), 'utf8');
  assert.match(account, /data-member-export="csv"/);
  assert.match(account, /data-member-export="xlsx"/);
  assert.match(script, /ipaceGetIdentityToken/);
  assert.match(script, /Authorization: 'Bearer '/);
  assert.match(script, /\/api\/member-export\?format=/);
  assert.match(main, /case "\/api\/member-export"/);
});
