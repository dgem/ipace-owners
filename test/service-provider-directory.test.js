const assert = require('node:assert/strict');
const { readFileSync } = require('node:fs');
const test = require('node:test');

test('the Jaguar EV service-provider lookup has structured, searchable UK records', function () {
  const directory = JSON.parse(readFileSync('src/assets/data/jaguar-uk-service-providers.json', 'utf8'));

  assert.equal(directory.source.name, 'Jaguar UK Find a Retailer');
  assert.match(directory.source.url, /^https:\/\/www\.jaguar\.com\//);
  assert.ok(directory.providers.length >= 50);
  directory.providers.forEach(function (provider) {
    assert.match(provider.id, /^[A-Z0-9]+$/);
    assert.ok(provider.name);
    assert.match(provider.postcode, /^[A-Z0-9 ]+$/);
    assert.equal(provider.electricVehicleService, true);
    assert.equal(typeof provider.authorisedRepairer, 'boolean');
    assert.equal(typeof provider.electricVehicleBatteryRepair, 'boolean');
  });
});
