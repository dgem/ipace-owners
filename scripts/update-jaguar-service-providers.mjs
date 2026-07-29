import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { format } from 'prettier';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const outputPath = path.join(root, 'src/assets/data/jaguar-uk-service-providers.json');
const sourcePage = 'https://www.jaguar.com/en-gb/jdx/national-dealer-locator.html';
const endpoint = 'https://retailerlocator.jaguarlandrover.com/dealers';

function directoryUrl(filter) {
  const url = new URL(endpoint);
  url.search = new URLSearchParams({
    requestMarketLocale: 'en_gb',
    brand: 'Jaguar',
    filter,
    radius: '100',
    unitOfMeasure: 'Miles',
    country: 'GB',
    fetchOpeningTimes: 'false',
  }).toString();
  return url;
}

async function fetchDirectory(filter) {
  const response = await fetch(directoryUrl(filter), {
    headers: { Accept: 'application/json' },
  });
  if (!response.ok) {
    throw new Error(`Jaguar retailer locator returned ${response.status} for ${filter}`);
  }
  const data = await response.json();
  if (!Array.isArray(data.dealers)) {
    throw new Error(`Jaguar retailer locator returned no dealer list for ${filter}`);
  }
  return data.dealers;
}

function clean(value) {
  return String(value || '').trim().replace(/\s+/g, ' ');
}

function providerFromDealer(dealer, batteryRepairIds) {
  const address = dealer.address || {};
  return {
    id: clean(dealer.ciCode),
    name: clean(dealer.name),
    postcode: clean(address.postCode).toUpperCase(),
    town: clean(address.town),
    county: clean(address.county),
    addressLines: [address.line1, address.line2, address.line3].map(clean).filter(Boolean),
    authorisedRepairer: dealer.authorisedRepairer === true,
    bodyshop: dealer.bodyshop === true,
    electricVehicleService: dealer.electricVehicleService === true,
    electricVehicleBatteryRepair: batteryRepairIds.has(clean(dealer.ciCode)),
  };
}

const [evServiceDealers, batteryRepairDealers] = await Promise.all([
  fetchDirectory('evs'),
  fetchDirectory('rperf'),
]);
const batteryRepairIds = new Set(batteryRepairDealers.map((dealer) => clean(dealer.ciCode)));
const providers = evServiceDealers
  .map((dealer) => providerFromDealer(dealer, batteryRepairIds))
  .filter((provider) => provider.id && provider.name && provider.postcode)
  .sort((left, right) => left.name.localeCompare(right.name, 'en-GB') || left.postcode.localeCompare(right.postcode, 'en-GB'));

if (providers.length < 50) {
  throw new Error(`Jaguar retailer locator returned only ${providers.length} usable UK EV service providers`);
}

const output = {
  source: {
    name: 'Jaguar UK Find a Retailer',
    url: sourcePage,
    serviceFilters: ['Electric Vehicle Service', 'Electric Vehicle Battery Repair'],
    retrievedAt: new Date().toISOString(),
  },
  providers,
};

await mkdir(path.dirname(outputPath), { recursive: true });
await writeFile(outputPath, await format(JSON.stringify(output), { parser: 'json' }), 'utf8');
console.log(`Wrote ${providers.length} Jaguar EV service providers to ${path.relative(root, outputPath)}`);
