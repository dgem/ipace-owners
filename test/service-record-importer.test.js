const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const test = require('node:test');

const read = (path) => readFileSync(path, 'utf8');

test('service document importer remains local, review-gated and uses the protected record API', function () {
  const page = read('src/member/import-service-records.njk');
  const script = read('src/assets/js/service-record-importer.js');
  const layout = read('src/_includes/layouts/base.njk');

  assert.match(page, /data-auth-container/);
  assert.match(page, /Original files are never uploaded/);
  assert.match(page, /data-import-files/);
  assert.match(layout, /service-record-importer\.js/);
  assert.match(script, /crypto\.subtle\.digest\('SHA-256'/);
  assert.match(script, /pdfjs\/pdf\.min\.mjs/);
  assert.match(script, /Tesseract\.createWorker/);
  assert.match(script, /pdfPageImages/);
  assert.match(script, /page ' \+ \(index \+ 1\) \+ ' of '/);
  assert.match(script, /Submit reviewed record/);
  assert.match(script, /fetch\('\/api\/upsert-service-event'/);
  assert.match(script, /ipaceGetIdentityToken/);
  assert.doesNotMatch(script, /FormData\(.*file/i);
  assert.doesNotMatch(script, /Source retained locally:/);
  assert.doesNotMatch(script, /title: source/);
});

test('browser OCR assets are generated from pinned dependencies', function () {
  const packageJSON = read('package.json');
  const copier = read('scripts/copy-browser-vendors.mjs');
  assert.match(packageJSON, /"pdfjs-dist"/);
  assert.match(packageJSON, /"tesseract\.js"/);
  assert.match(copier, /eng\.traineddata\.gz/);
  assert.match(copier, /copyFileSync/);
});
