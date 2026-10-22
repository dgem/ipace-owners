(function () {
  'use strict';

  var dismissed = null;
  var loading = null;

  function token() {
    return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : Promise.resolve('');
  }

  function request(path, options) {
    return token().then(function (idToken) {
      if (!idToken) throw new Error('Sign in required');
      options = options || {};
      options.headers = Object.assign({ Authorization: 'Bearer ' + idToken }, options.headers || {});
      if (window.ipaceAuthHeaders) options.headers = window.ipaceAuthHeaders(options.headers);
      return fetch(path, options);
    }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok) throw new Error(data.error || 'Request failed');
        return data;
      });
    });
  }

  function enhance(element) {
    var id = element.dataset.notificationId;
    if (!id) return;
    if (dismissed && dismissed.indexOf(id) !== -1) {
      element.dataset.notificationDismissed = 'true';
      element.hidden = true;
      return;
    }
    delete element.dataset.notificationDismissed;
    if (!element.dataset.memberFeature) element.hidden = false;
    if (element.querySelector('[data-dismiss-member-notification]')) return;
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'member-notification__dismiss';
    button.dataset.dismissMemberNotification = '';
    button.textContent = 'Dismiss';
    button.setAttribute('aria-label', 'Dismiss this notice');
    button.addEventListener('click', function () {
      button.disabled = true;
      request('/api/member/notifications', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: id })
      }).then(function () {
        dismissed = dismissed || [];
        if (dismissed.indexOf(id) === -1) dismissed.push(id);
        element.dataset.notificationDismissed = 'true';
        element.hidden = true;
      }).catch(function () {
        button.disabled = false;
        button.textContent = 'Try again';
      });
    });
    element.appendChild(button);
  }

  function apply() {
    document.querySelectorAll('[data-member-notification][data-notification-id]').forEach(enhance);
  }

  function load(force) {
    if (dismissed && !force) { apply(); return Promise.resolve(dismissed); }
    if (loading && !force) return loading;
    if (!document.querySelector('[data-member-notification]')) return Promise.resolve([]);
    loading = request('/api/member/notifications').then(function (data) {
      dismissed = data.dismissed || [];
      loading = null;
      apply();
      return dismissed;
    }).catch(function () {
      dismissed = [];
      loading = null;
      apply();
      return dismissed;
    });
    return loading;
  }

  window.ipaceRefreshMemberNotifications = function () { return load(false); };
  document.addEventListener('member:data', function () { load(false); });
  document.addEventListener('identity:login', function () { load(true); });
  document.addEventListener('identity:logout', function () { dismissed = null; loading = null; });
}());
