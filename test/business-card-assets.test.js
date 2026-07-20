const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');

test('hero business-card variant preserves approved imagery, QR and copy', function () {
  const svg = fs.readFileSync(
    path.join(root, 'public/images/ipace-owners-card-front-hero.svg'),
    'utf8'
  );
  const png = fs.readFileSync(path.join(root, 'public/images/ipace-owners-card-front-hero.png'));

  assert.match(svg, /viewBox="0 0 850 550"/);
  assert.match(svg, /href="ipace-hero\.png"/);
  assert.match(svg, /href="ipace-owners-qr\.svg"/);
  assert.match(svg, /H447\/H570/);
  assert.match(svg, /Battery issues\?/);
  assert.match(svg, /Join us to help get a fair deal for all\./);
  assert.match(svg, /Free to join\. It takes less than a minute\./);
  assert.match(svg, /iPace-Owners\.org/);
  assert.ok(png.length > 100000, 'expected a rendered print-resolution PNG');
});
