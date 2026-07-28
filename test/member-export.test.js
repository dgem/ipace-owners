const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');

test('member account offers authenticated CSV and Excel exports', () => {
  const account = fs.readFileSync(path.join(root, 'src/member/account.njk'), 'utf8');
  const update = fs.readFileSync(path.join(root, 'src/updates/member-data-export.md'), 'utf8');
  const privacy = fs.readFileSync(path.join(root, 'src/privacy.md'), 'utf8');
  const samplePath = path.join(root, 'public/downloads/sample-ipace-owner-data.xlsx');
  const script = fs.readFileSync(path.join(root, 'src/assets/js/member-export.js'), 'utf8');
  const main = fs.readFileSync(path.join(root, 'functions/firebase-go/main.go'), 'utf8');
  assert.match(account, /data-member-export="csv"/);
  assert.match(account, /data-member-export="xlsx"/);
  assert.match(account, /href="\/member\/dashboard\/"[^>]*>Vehicle Data<\/a>/);
  assert.match(account, /href="\/downloads\/sample-ipace-owner-data\.xlsx"/);
  assert.match(update, /feature requested by a member/i);
  assert.match(update, /fictional; it contains no member data/i);
  assert.match(privacy, /Exports are generated on demand from your authenticated member snapshot/);
  assert.match(privacy, /Information Commissioner's Office/);
  assert.equal(fs.existsSync(samplePath), true);
  assert.equal(fs.readFileSync(samplePath, null).subarray(0, 2).toString(), 'PK');
  assert.match(script, /ipaceGetIdentityToken/);
  assert.match(script, /Authorization: 'Bearer '/);
  assert.match(script, /\/api\/member-export\?format=/);
  assert.match(main, /case "\/api\/member-export"/);
});
