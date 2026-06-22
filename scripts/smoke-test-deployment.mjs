const rawBaseUrl = process.env.SMOKE_BASE_URL;

if (!rawBaseUrl) {
  throw new Error('SMOKE_BASE_URL is required');
}

const baseUrl = new URL(rawBaseUrl);
baseUrl.pathname = '/';
baseUrl.search = '';
baseUrl.hash = '';

function isAllowedSmokeHost(hostname) {
  return (
    hostname === 'localhost' ||
    hostname === '127.0.0.1' ||
    hostname === 'ipace-owners.org' ||
    hostname === 'www.ipace-owners.org' ||
    hostname.endsWith('.ipace-owners.org') ||
    hostname.endsWith('.web.app') ||
    hostname.endsWith('.firebaseapp.com')
  );
}

if (!isAllowedSmokeHost(baseUrl.hostname)) {
  throw new Error(`Refusing to smoke test unsupported host: ${baseUrl.hostname}`);
}

function url(path) {
  return new URL(path, baseUrl).toString();
}

async function fetchText(path) {
  const target = url(path);
  const res = await fetch(target, { redirect: 'follow' });
  if (!res.ok) {
    throw new Error(`${path} returned ${res.status} from ${target}`);
  }
  return res.text();
}

function excerpt(value) {
  return value
    .replace(/<script\b[^>]*>[\s\S]*?<\/script>/gi, '')
    .replace(/<style\b[^>]*>[\s\S]*?<\/style>/gi, '')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 500);
}

function assertIncludes(value, expected, label) {
  if (!value.includes(expected)) {
    throw new Error(`${label} did not include ${expected}`);
  }
}

function assertIncludesAny(value, expectedValues, label) {
  if (!expectedValues.some((expected) => value.includes(expected))) {
    throw new Error(`${label} did not include any of ${expectedValues.join(', ')}. Fetched ${baseUrl.toString()} and saw: ${excerpt(value)}`);
  }
}

function assertNotIncludes(value, unexpected, label) {
  if (value.includes(unexpected)) {
    throw new Error(`${label} unexpectedly included ${unexpected}`);
  }
}

async function main() {
  console.log(`Running smoke tests for ${baseUrl.toString()}`);

  const home = await fetchText('/');
  assertIncludesAny(home, ['i-Pace Owners', 'I-PACE Owners', 'Owners working together'], 'home page');

  const account = await fetchText('/account/');
  assertIncludes(account, 'data-magic-link-form', 'account page');
  assertNotIncludes(account, 'data-identity-open', 'account page');

  const vehicle = await fetchText('/submit-vehicle-data/');
  assertIncludes(vehicle, 'data-magic-link-form', 'vehicle page');

  const identityJs = await fetchText('/assets/js/identity.js');
  assertIncludes(identityJs, '/api/send-magic-link', 'identity.js');
  assertNotIncludes(identityJs, 'identity.open(', 'identity.js');

  const magicLinkPreflight = await fetch(url('/api/send-magic-link'), {
    method: 'OPTIONS',
  });
  if (magicLinkPreflight.status !== 204) {
    throw new Error(`send-magic-link preflight returned ${magicLinkPreflight.status}, expected 204`);
  }

  const vehicleUnauthenticated = await fetch(url('/api/submit-vehicle-basics'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ registration: 'SMOKE TEST' }),
  });
  if (vehicleUnauthenticated.status !== 401) {
    throw new Error(`submit-vehicle-basics unauthenticated returned ${vehicleUnauthenticated.status}, expected 401`);
  }

  const sohUnauthenticated = await fetch(url('/api/submit-soh'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ vehicleId: 'smoke', soh: '90', sohDate: '2026-06-22', sohSource: 'diagnostic-app' }),
  });
  if (sohUnauthenticated.status !== 401) {
    throw new Error(`submit-soh unauthenticated returned ${sohUnauthenticated.status}, expected 401`);
  }

  const publicStats = await fetch(url('/api/public-stats'));
  if (!publicStats.ok) {
    throw new Error(`public-stats returned ${publicStats.status}, expected 200`);
  }

  console.log(`Smoke tests passed for ${baseUrl.toString()}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
