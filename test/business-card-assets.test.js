const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');

function withoutEmbeddedAssets(svg) {
  return svg.replace(/data:[^;]+;base64,[A-Za-z0-9+/=]+/g, 'embedded-asset');
}

test('original business-card front uses current H570 copy and a transparent QR treatment', function () {
  const svg = fs.readFileSync(
    path.join(root, 'public/images/ipace-owners-card-front.svg'),
    'utf8'
  );
  const png = fs.readFileSync(path.join(root, 'public/images/ipace-owners-card-front.png'));

  assert.match(svg, />H570 Battery issues\?<\/text>/);
  assert.doesNotMatch(withoutEmbeddedAssets(svg), /H441|H447/);
  assert.match(svg, /data:image\/svg\+xml;base64,[A-Za-z0-9+/=]+"[^>]+opacity="\.82"/);
  assert.ok(png.length > 100000, 'expected a rendered print-resolution PNG');
});

test('hero business-card variant preserves approved imagery, QR and copy', function () {
  const svg = fs.readFileSync(
    path.join(root, 'public/images/ipace-owners-card-front-hero.svg'),
    'utf8'
  );
  const png = fs.readFileSync(path.join(root, 'public/images/ipace-owners-card-front-hero.png'));

  assert.match(svg, /viewBox="0 0 850 550"/);
  assert.match(svg, /href="data:image\/png;base64,[A-Za-z0-9+/=]+" x="-145"/);
  assert.match(svg, /href="data:image\/svg\+xml;base64,[A-Za-z0-9+/=]+"/);
  assert.doesNotMatch(svg, /href="ipace-(?:hero\.png|owners-qr\.svg)"/);
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
  assert.match(svg, /data:image\/svg\+xml;base64,[A-Za-z0-9+/=]+" x="36" y="394"[^>]+opacity="\.82"/);
  assert.doesNotMatch(svg, /white-qr|url\(#white-qr\)/);
  assert.doesNotMatch(svg, /width="836" height="536"[^>]+stroke="#fff"/);
  assert.doesNotMatch(svg, /clipPath|clip-path|rx="18"/);
  assert.ok(png.length > 100000, 'expected a rendered print-resolution PNG');
});

test('hero business-card back matches the photographic front and concise H57x positioning', function () {
  const svg = fs.readFileSync(
    path.join(root, 'public/images/ipace-owners-card-back-hero.svg'),
    'utf8'
  );
  const png = fs.readFileSync(path.join(root, 'public/images/ipace-owners-card-back-hero.png'));

  assert.match(svg, /href="data:image\/png;base64,[A-Za-z0-9+/=]+" x="-145"/);
  assert.doesNotMatch(svg, /href="ipace-hero\.png"/);
  assert.match(svg, /I-PACE owners working together/);
  assert.match(svg, /working together for fair outcomes<\/text>/);
  assert.match(svg, /Engaging constructively with Jaguar\./);
  assert.match(svg, /H57X RECALL SERIES/);
  assert.match(svg, /Traction battery faults/i);
  assert.match(svg, /reduced range and performance/i);
  assert.match(svg, /Fair, consistent support/i);
  assert.doesNotMatch(withoutEmbeddedAssets(svg), /ipace-owners-qr|H441|H448|H570|H571|H572/);
  assert.ok(png.length > 100000, 'expected a rendered print-resolution back PNG');
});
