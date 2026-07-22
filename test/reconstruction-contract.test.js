const fs = require('node:fs');
const path = require('node:path');
const test = require('node:test');
const assert = require('node:assert/strict');

const root = path.join(__dirname, '..');
const promptFilenamePattern = /^\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$/;

function read(relativePath) {
  return fs.readFileSync(path.join(root, relativePath), 'utf8');
}

function exists(relativePath) {
  return fs.existsSync(path.join(root, relativePath));
}

test('clean-room contract tracks the complete maintained prompt set', function () {
  const prompts = fs
    .readdirSync(path.join(root, 'prompts'))
    .filter((name) => name.endsWith('.md'))
    .sort();
  const promptNumbers = prompts.map((name) => {
    assert.match(name, promptFilenamePattern, `${name} must match xx-name.md`);
    return Number(name.slice(0, 2));
  });

  assert.equal(prompts[0], '00-original-project-prompt.md');
  assert.ok(prompts.includes('20-clean-room-reconstruction-contract.md'));
  assert.deepEqual(
    promptNumbers,
    Array.from({ length: promptNumbers.at(-1) + 1 }, (_, index) => index)
  );
});

test('clean-room route inventory maps to source pages and Hosting redirects', function () {
  const contract = read('prompts/20-clean-room-reconstruction-contract.md');
  const firebase = read('firebase.json');
  const routes = [
    ['/', 'src/index.njk'],
    ['/about/', 'src/about.md'],
    ['/faq/', 'src/faq.njk'],
    ['/join/', 'src/join.njk'],
    ['/contact/', 'src/contact.md'],
    ['/privacy/', 'src/privacy.md'],
    ['/terms/', 'src/terms.md'],
    ['/methodology/', 'src/methodology.md'],
    ['/evidence-dashboard/', 'src/evidence-dashboard.njk'],
    ['/updates/', 'src/updates.njk'],
    ['/member/dashboard/', 'src/member/dashboard.njk'],
    ['/member/account/', 'src/member/account.njk'],
    ['/member/submit-vehicle-data/', 'src/member/submit-vehicle-data.njk'],
    ['/admin/review-queue/', 'src/admin/review-queue.njk'],
    ['/admin/outreach/', 'src/admin/outreach.njk'],
    ['/404.html', 'src/404.njk'],
  ];

  for (const [route, source] of routes) {
    assert.ok(exists(source), `missing source for ${route}: ${source}`);
    assert.ok(contract.includes(route), `clean-room contract omits ${route}`);
  }

  for (const legacyRoute of ['/account', '/submit-vehicle-data']) {
    assert.ok(firebase.includes(`"source": "${legacyRoute}"`));
    assert.ok(contract.includes(`${legacyRoute}/**`));
  }
  assert.match(firebase, /"destination": "\/404\.html"/);
});

test('clean-room API and Firestore inventories match implemented handlers', function () {
  const contract = read('prompts/20-clean-room-reconstruction-contract.md');
  const architecture = read('prompts/09-architecture-overview.md');
  const readme = read('README.md');
  const main = read('functions/firebase-go/main.go');
  const goSource = fs
    .readdirSync(path.join(root, 'functions/firebase-go'))
    .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
    .map((name) => read(path.join('functions/firebase-go', name)))
    .join('\n');
  const routes = Array.from(main.matchAll(/case "(\/api\/[^"]+)"/g), (match) => match[1]);
  const collections = [
    'joinSubmissions',
    'members',
    'vehicles',
    'batteryReadings',
    'serviceEvents',
    'memberSnapshots',
  ];

  for (const route of routes) {
    assert.ok(main.includes(`case "${route}"`), `Api router omits ${route}`);
    assert.ok(architecture.includes(route), `architecture prompt omits ${route}`);
    assert.ok(contract.includes(route), `clean-room contract omits ${route}`);
    assert.ok(readme.includes(route.slice('/api/'.length)), `README omits ${route}`);
  }
  for (const collection of collections) {
    assert.ok(goSource.includes(`Collection("${collection}")`), `Go source omits ${collection}`);
    assert.ok(contract.includes(`\`${collection}\``), `clean-room contract omits ${collection}`);
  }

  assert.match(goSource, /const publicStatsSchemaVersion = 5/);
  assert.match(contract, /`schemaVersion: 5`/);
  assert.match(read('src/assets/js/public-stats.js'), /\/api\/public-stats\?v=5/);
  assert.doesNotMatch(contract, /JSON\/form|rate-limiting behaviour described/);
});

test('clean-room configuration and preservation-critical asset inventories stay reproducible', function () {
  const contract = read('prompts/20-clean-room-reconstruction-contract.md');
  const launch = read('prompts/19-launch-readiness.md');
  const requiredConfig = [
    'FIREBASE_WEB_API_KEY',
    'FIREBASE_AUTH_DOMAIN',
    'FIREBASE_PROJECT_ID',
    'FIREBASE_APP_ID',
    'FIREBASE_STORAGE_BUCKET',
    'FIRESTORE_DATABASE_ID',
    'SNAPSHOT_BUCKET',
    'VIN_PEPPER',
    'ALLOWED_ORIGINS',
    'FIREBASE_EMAIL_CONTINUE_URL',
    'FIREBASE_EMAIL_LINK_DOMAIN',
    'RESEND_API_KEY',
    'RESEND_FROM',
    'RESEND_REPLY_TO',
    'RESEND_ASSET_BASE_URL',
  ];
  const assets = [
    'public/favicon.png',
    'public/images/ipace-hero.png',
    'public/images/ipace-owners-logo.svg',
    'public/images/ipace-owners-logo.png',
    'public/images/ipace-owners-qr.svg',
    'public/images/ipace-owners-card-front.svg',
    'public/images/ipace-owners-card-front.png',
    'public/images/ipace-owners-card-back.svg',
    'public/images/ipace-owners-card-back.png',
    'public/images/ipace-owners-card-front-hero.svg',
    'public/images/ipace-owners-card-front-hero.png',
    'public/images/ipace-owners-card-back-hero.svg',
    'public/images/ipace-owners-card-back-hero.png',
  ];

  for (const name of requiredConfig) assert.ok(contract.includes(`\`${name}\``), `contract omits ${name}`);
  for (const asset of assets) {
    assert.ok(exists(asset), `missing preservation-critical asset ${asset}`);
    const documented = contract.includes(asset) || launch.includes(asset);
    assert.ok(documented, `prompts omit preservation-critical asset ${asset}`);
  }
  assert.doesNotMatch(contract, /printable business-card PDF/);
});

test('maintainer documentation does not regress to superseded project descriptions', function () {
  const readme = read('README.md');
  const agents = read('AGENTS.md');
  const prTemplate = read('.github/pull_request_template.md');
  const maintainedPrompts = fs
    .readdirSync(path.join(root, 'prompts'))
    .filter((name) => /^\d{2}-.*\.md$/.test(name) && !name.startsWith('00-'))
    .map((name) => read(path.join('prompts', name)))
    .join('\n');
  const browserModules = fs
    .readdirSync(path.join(root, 'src/assets/js'))
    .filter((name) => name.endsWith('.js'));

  assert.doesNotMatch(readme, /\(coming soon\)|Three files:|`01-` through `14-`/);
  assert.doesNotMatch(readme, /Member-gated placeholder|Admin-gated placeholder/);
  assert.match(readme, /live since 17 July 2026/);
  for (const module of browserModules) {
    assert.ok(readme.includes(`\`${module}\``), `README omits browser module ${module}`);
  }
  assert.match(agents, /prompt-drift check is mandatory whenever raising or updating a PR/i);
  assert.match(agents, /data-auth-container/);
  assert.match(agents, /data-admin-container/);
  assert.doesNotMatch(prTemplate, /docs\/architecture\.md/);
  assert.doesNotMatch(maintainedPrompts, /public\/favicon\.svg|H447|H441\/H570/);
});

test('documented Make commands remain available', function () {
  const makefile = read('Makefile');
  const documentedCommands = [
    read('README.md'),
    read('AGENTS.md'),
    read('prompts/02-foundation-static-site.md'),
    read('prompts/17-operations-ci-and-troubleshooting.md'),
    read('prompts/19-launch-readiness.md'),
    read('prompts/20-clean-room-reconstruction-contract.md'),
  ].join('\n');
  const targets = new Set(Array.from(makefile.matchAll(/^([a-zA-Z0-9_.-]+):/gm), (match) => match[1]));
  const mentioned = new Set(
    Array.from(documentedCommands.matchAll(/`make\s+([a-z][a-z0-9-]*)/g), (match) => match[1])
      .filter((name) => !name.endsWith('-'))
  );

  for (const target of mentioned) {
    assert.ok(targets.has(target), `documentation refers to missing Make target ${target}`);
  }
});
