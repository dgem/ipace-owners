const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');

const root = path.join(__dirname, '..');

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8');
}

test('maintained prompts form a contiguous 00-20 reconstruction set', function () {
  const promptFiles = fs
    .readdirSync(path.join(root, 'prompts'))
    .filter(function (name) {
      return /^\d{2}-.*\.md$/.test(name);
    })
    .sort();

  assert.deepEqual(
    promptFiles.map(function (name) {
      return Number(name.slice(0, 2));
    }),
    Array.from({ length: 21 }, function (_, index) {
      return index;
    })
  );
});

test('architecture documents enumerate every routed API endpoint', function () {
  const apiSource = read('functions/firebase-go/main.go');
  const architecture = read('prompts/09-architecture-overview.md');
  const reconstruction = read('prompts/20-clean-room-reconstruction-contract.md');
  const readme = read('README.md');
  const routes = Array.from(apiSource.matchAll(/case "(\/api\/[^"]+)"/g), function (match) {
    return match[1];
  });

  assert.ok(routes.length > 0, 'expected API routes in the Go router');
  routes.forEach(function (route) {
    assert.ok(architecture.includes(route), 'architecture prompt is missing ' + route);
    assert.ok(reconstruction.includes(route), 'reconstruction contract is missing ' + route);
    assert.ok(readme.includes(route.slice('/api/'.length)), 'README is missing ' + route);
  });
});

test('README inventories every browser JavaScript module', function () {
  const readme = read('README.md');
  const modules = fs
    .readdirSync(path.join(root, 'src/assets/js'))
    .filter(function (name) {
      return name.endsWith('.js');
    });

  modules.forEach(function (name) {
    assert.ok(readme.includes('`' + name + '`'), 'README is missing ' + name);
  });
});

test('preservation-critical visual assets exist and are named in the reconstruction contract', function () {
  const reconstruction = read('prompts/20-clean-room-reconstruction-contract.md');
  const assets = [
    'public/favicon.png',
    'public/images/ipace-hero.png',
    'public/images/ipace-owners-logo.png',
    'public/images/ipace-owners-logo.svg',
    'public/images/ipace-owners-qr.svg',
    'public/images/ipace-owners-card-front.png',
    'public/images/ipace-owners-card-front.svg',
    'public/images/ipace-owners-card-back.png',
    'public/images/ipace-owners-card-back.svg',
    'public/images/ipace-owners-card-front-hero.png',
    'public/images/ipace-owners-card-front-hero.svg',
    'public/images/ipace-owners-card-back-hero.png',
    'public/images/ipace-owners-card-back-hero.svg',
  ];

  assets.forEach(function (asset) {
    assert.ok(fs.existsSync(path.join(root, asset)), 'missing preservation asset ' + asset);
    assert.ok(reconstruction.includes(asset), 'reconstruction contract is missing ' + asset);
  });
});

test('maintained documentation contains no known superseded references', function () {
  const maintained = [
    'README.md',
    'AGENTS.md',
    '.github/pull_request_template.md',
  ].concat(
    fs
      .readdirSync(path.join(root, 'prompts'))
      .filter(function (name) {
        return /^\d{2}-.*\.md$/.test(name) && !name.startsWith('00-');
      })
      .map(function (name) {
        return path.join('prompts', name);
      })
  );
  const text = maintained.map(read).join('\n');

  assert.doesNotMatch(text, /docs\/architecture\.md/);
  assert.doesNotMatch(text, /public\/favicon\.svg/);
  assert.doesNotMatch(text, /H447/);
  assert.doesNotMatch(read('README.md'), /coming soon/i);
});

test('pull-request guidance makes prompt-drift review mandatory', function () {
  const agents = read('AGENTS.md');
  const template = read('.github/pull_request_template.md');

  assert.match(agents, /prompt-drift check is mandatory whenever raising or updating a PR/);
  assert.match(template, /Prompt and documentation alignment/);
  assert.match(template, /updated prompts in prompts\//);
});
