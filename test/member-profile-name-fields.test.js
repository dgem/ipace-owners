const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

test('member contact profile separates forenames and family name', function () {
  const account = fs.readFileSync(path.join(__dirname, '../src/member/account.njk'), 'utf8');
  const profile = fs.readFileSync(path.join(__dirname, '../src/assets/js/member-profile.js'), 'utf8');
  const backend = fs.readFileSync(path.join(__dirname, '../functions/firebase-go/member_profile.go'), 'utf8');

  assert.match(account, /name="forenames"[^>]*autocomplete="given-name"/);
  assert.match(account, /name="familyName"[^>]*autocomplete="family-name"/);
  assert.match(profile, /'forenames', 'familyName'/);
  assert.match(backend, /json:"forenames" firestore:"forenames"/);
  assert.match(backend, /json:"familyName" firestore:"familyName"/);
  assert.match(backend, /splitLegacyMemberName/);
});
