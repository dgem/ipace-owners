(function () {
  'use strict';

  var state = null;
  var loading = null;
  var generation = 0;

  function token() {
    return Promise.resolve(window.ipaceIdentityReadyPromise).then(function () {
      return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : '';
    });
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
    if (loading && !force) return loading;
    if (state && !force) return Promise.resolve(state);
    var currentGeneration = ++generation;
    loading = token().then(function (idToken) {
      var headers = idToken ? { Authorization: 'Bearer ' + idToken } : {};
      if (window.ipaceAuthHeaders) headers = window.ipaceAuthHeaders(headers);
      return fetch('/api/member/features', { headers: headers }).then(function (response) {
        if (!response.ok) return {};
        return response.json().then(function (data) { return data.features || {}; });
      });
    }).catch(function () {
      return {};
    }).then(function (features) {
      if (currentGeneration !== generation) return state || {};
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

  document.addEventListener('member:data', function () { load(true); });
  document.addEventListener('identity:ready', function () { load(false); });
  document.addEventListener('identity:login', function () { load(true); });
  document.addEventListener('identity:logout', function () { generation++; state = null; loading = null; apply(); load(false); });
  apply();
  load(false);
}());
