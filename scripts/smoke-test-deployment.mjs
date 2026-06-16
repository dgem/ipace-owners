const rawBaseUrl = process.env.SMOKE_BASE_URL;

if (!rawBaseUrl) {
  throw new Error('SMOKE_BASE_URL is required');
}

const baseUrl = new URL(rawBaseUrl);
baseUrl.pathname = '/';
baseUrl.search = '';
baseUrl.hash = '';

function url(path) {
  return new URL(path, baseUrl).toString();
}

async function fetchText(path) {
  const res = await fetch(url(path), { redirect: 'follow' });
  if (!res.ok) {
    throw new Error(`${path} returned ${res.status}`);
  }
  return res.text();
}

function assertIncludes(value, expected, label) {
  if (!value.includes(expected)) {
    throw new Error(`${label} did not include ${expected}`);
  }
}

function assertNotIncludes(value, unexpected, label) {
  if (value.includes(unexpected)) {
    throw new Error(`${label} unexpectedly included ${unexpected}`);
  }
}

async function main() {
  const home = await fetchText('/');
  assertIncludes(home, 'i-Pace Owners', 'home page');

  const account = await fetchText('/account/');
  assertIncludes(account, 'data-magic-link-form', 'account page');
  assertNotIncludes(account, 'data-identity-open', 'account page');

  const vehicle = await fetchText('/submit-vehicle-data/');
  assertIncludes(vehicle, 'data-magic-link-form', 'vehicle page');

  const identityJs = await fetchText('/assets/js/identity.js');
  assertIncludes(identityJs, '/.netlify/functions/send-magic-link', 'identity.js');
  assertNotIncludes(identityJs, 'identity.open(', 'identity.js');

  const badMagicLink = await fetch(url('/.netlify/functions/send-magic-link'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email: 'not-an-email' }),
  });
  if (badMagicLink.status !== 400) {
    throw new Error(`send-magic-link invalid email returned ${badMagicLink.status}, expected 400`);
  }

  const vehicleUnauthenticated = await fetch(url('/.netlify/functions/submit-vehicle-basics'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ registration: 'SMOKE TEST' }),
  });
  if (vehicleUnauthenticated.status !== 401) {
    throw new Error(`submit-vehicle-basics unauthenticated returned ${vehicleUnauthenticated.status}, expected 401`);
  }

  console.log(`Smoke tests passed for ${baseUrl.toString()}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});
