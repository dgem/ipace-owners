(function () {
  'use strict';
  var form = document.querySelector('[data-member-profile-form]');
  if (!form) return;
  var status = form.querySelector('[data-member-profile-status]');
  var represent = form.elements.represent;
  var addresses = [];

  function uk() {
    return /^(GB|GBR|UK|UNITED KINGDOM|ENGLAND|SCOTLAND|WALES|NORTHERN IRELAND)$/i.test(form.elements.country.value.trim());
  }

  function updateEligibility() {
    if (!represent) return;
    var eligible = uk();
    represent.disabled = !eligible;
    if (!eligible) represent.checked = false;
    form.querySelector('[data-representation-hint]').textContent = eligible
      ? 'Currently available only with a complete UK address. Identity verification is a separate review.'
      : 'Registering interest in legal options is currently available only to members with a UK address.';
  }

  function api(url, options) {
    return window.ipaceGetIdentityToken().then(function (token) {
      if (!token) throw new Error('Sign in to manage your profile.');
      options = options || {};
      options.headers = Object.assign({}, options.headers || {}, { Authorization: 'Bearer ' + token });
      return fetch(url, options);
    }).then(function (response) {
      return response.json().catch(function () {
        throw new Error('Profile service is unavailable. Please try again later.');
      }).then(function (data) {
        if (!response.ok) throw new Error(data.error || 'Request failed');
        return data;
      });
    });
  }

  function load() {
    if (!window.ipaceIdentityUser) return;
    api('/api/member-profile').then(function (data) {
      Object.keys(data.profile || {}).forEach(function (key) {
        if (!form.elements[key]) return;
        if (form.elements[key].type === 'checkbox') form.elements[key].checked = !!data.profile[key];
        else form.elements[key].value = data.profile[key] || '';
      });
      var wording = form.querySelector('[data-representation-wording]');
      if (wording && data.representationWording) wording.textContent = data.representationWording;
      updateEligibility();
    }).catch(function (error) { status.textContent = error.message; });
  }

  form.elements.country.addEventListener('input', updateEligibility);
  form.querySelector('[data-address-lookup]').addEventListener('click', function () {
    status.textContent = 'Looking up addresses…';
    var query = form.querySelector('[data-address-query]').value.trim() || form.elements.postalCode.value.trim();
    api('/api/member-address-lookup?query=' + encodeURIComponent(query) + '&country=' + encodeURIComponent(form.elements.country.value)).then(function (data) {
      addresses = data.suggestions || [];
      var select = form.querySelector('[data-address-select]');
      select.replaceChildren(new Option('Choose an address', ''));
      addresses.forEach(function (address, index) {
        select.add(new Option(address.suggestion, String(index)));
      });
      form.querySelector('[data-address-results]').hidden = !addresses.length;
      status.textContent = addresses.length ? 'Choose an address, then check the details.' : 'No addresses found. Enter your address manually.';
    }).catch(function (error) { status.textContent = error.message; });
  });
  form.querySelector('[data-address-select]').addEventListener('change', function (event) {
    if (event.target.value === '') return;
    var selected = addresses[Number(event.target.value)];
    if (!selected) return;
    status.textContent = 'Loading address…';
    api('/api/member-address-lookup?id=' + encodeURIComponent(selected.id)).then(function (data) {
      var address = data.address;
      form.elements.addressLine1.value = address.line_1 || '';
      form.elements.addressLine2.value = address.line_2 || '';
      form.elements.city.value = address.post_town || '';
      form.elements.region.value = address.county || '';
      form.elements.postalCode.value = address.postcode || form.elements.postalCode.value;
      if (address.country) form.elements.country.value = address.country;
      updateEligibility();
      status.textContent = 'Check the address before saving.';
    }).catch(function (error) { status.textContent = error.message; });
  });

  form.addEventListener('submit', function (event) {
    event.preventDefault();
    var payload = {};
    ['name', 'phone', 'addressLine1', 'addressLine2', 'city', 'region', 'postalCode', 'country'].forEach(function (key) {
      payload[key] = form.elements[key].value;
    });
    if (represent) payload.represent = represent.checked && !represent.disabled;
    var button = form.querySelector('[type="submit"]');
    button.disabled = true;
    status.textContent = 'Saving…';
    api('/api/member-profile', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
      .then(function () { status.textContent = 'Contact details saved.'; })
      .catch(function (error) { status.textContent = error.message; })
      .finally(function () { button.disabled = false; });
  });

  document.addEventListener('identity:login', load);
  document.addEventListener('identity:ready', load);
  if (window.ipaceIdentityUser) load();
})();
