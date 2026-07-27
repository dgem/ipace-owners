const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');

test('known issues page uses summaries and official source links without publishing supplied PDFs', () => {
  const page = fs.readFileSync(path.join(root, 'src/known-issues.njk'), 'utf8');
  assert.match(page, /SSM76024/);
  assert.match(page, /SSM76062/);
  assert.match(page, /JLRTB02087/);
  assert.match(page, /JLRTB02109/);
  assert.match(page, /JLRTB02043V3/);
  const sourceHosts = new Set(
    page.split('href="').slice(1)
      .map((part) => part.split('"')[0])
      .filter((href) => href.startsWith('https://'))
      .map((href) => new URL(href).hostname),
  );
  assert.ok(sourceHosts.has('topix.jaguar.jlrext.com'));
  assert.ok(sourceHosts.has('topix.landrover.jlrext.com'));
  assert.ok(sourceHosts.has('static.nhtsa.gov'));
  assert.match(page, /not recalls/);
  assert.match(page, /do not prove that a particular VIN is affected/);
  assert.doesNotMatch(page, /docs\/topix|\/documents\/.*\.pdf/i);
});

test('known issues is discoverable in public navigation and footer', () => {
  const navigation = JSON.parse(fs.readFileSync(path.join(root, 'src/_data/navigation.json'), 'utf8'));
  assert.ok(navigation.public.some((item) => item.url === '/known-issues/'));
  const footer = fs.readFileSync(path.join(root, 'src/_includes/partials/footer.njk'), 'utf8');
  assert.match(footer, /href="\/known-issues\/"/);
});
