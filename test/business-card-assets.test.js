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
  assert.match(svg, /href="ipace-hero\.png" x="-145"/);
  assert.match(svg, /href="ipace-owners-qr\.svg"/);
  assert.match(svg, />H570 battery issues\?<\/text>/);
  assert.doesNotMatch(svg, />I-PACE OWNERS<\/text>/);
  assert.doesNotMatch(svg, /width="187" height="34" rx="17"/);
  assert.match(svg, /Join us to help get a fair deal for all\./);
  assert.match(svg, /Free to join, takes less than a min\./);
  assert.match(svg, /iPace-Owners\.org/);
  assert.doesNotMatch(svg, /feDropShadow|text-shadow/);
  assert.match(svg, /text-rendering="geometricPrecision"/);
  assert.doesNotMatch(svg, /id="footer-shade"/);
  assert.doesNotMatch(svg, /width="140" height="140"[^>]+fill="#fff"/);
  assert.match(svg, /href="ipace-owners-qr\.svg" x="36" y="394"/);
  assert.doesNotMatch(svg, /width="836" height="536"[^>]+stroke="#fff"/);
  assert.doesNotMatch(svg, /clipPath|clip-path|rx="18"/);
  assert.ok(png.length > 100000, 'expected a rendered print-resolution PNG');
});
