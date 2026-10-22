(function () {
  'use strict';

  var state = null;
  var loading = null;

  function token() {
    return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : Promise.resolve('');
  }

  function apply() {
    document.querySelectorAll('[data-member-feature]').forEach(function (element) {
      element.hidden = !(state && state[element.dataset.memberFeature]) || element.dataset.notificationDismissed === 'true';
    });
    document.querySelectorAll('[data-member-feature-denied]').forEach(function (element) {
      element.hidden = state === null || !!state[element.dataset.memberFeatureDenied];
    });
  }

  function load(force) {
    if (state && !force) return Promise.resolve(state);
    if (loading && !force) return loading;
    loading = token().then(function (idToken) {
      if (!idToken) return {};
      var headers = { Authorization: 'Bearer ' + idToken };
      if (window.ipaceAuthHeaders) headers = window.ipaceAuthHeaders(headers);
      return fetch('/api/member/features', { headers: headers }).then(function (response) {
        if (!response.ok) return {};
        return response.json().then(function (data) { return data.features || {}; });
      });
    }).catch(function () {
      return {};
    }).then(function (features) {
      state = features;
      loading = null;
      apply();
      document.dispatchEvent(new CustomEvent('member:features', { detail: { features: state } }));
      return state;
    });
    return loading;
  }

  window.ipaceMemberFeatureEnabled = function (name) {
    return load(false).then(function (features) { return features[name] === true; });
  };
  window.ipaceRefreshMemberFeatures = function () {
    apply();
    return load(false);
  };

  document.addEventListener('member:data', function () { load(false); });
  document.addEventListener('identity:login', function () { load(true); });
  document.addEventListener('identity:logout', function () { state = {}; loading = null; apply(); });
  apply();
}());
