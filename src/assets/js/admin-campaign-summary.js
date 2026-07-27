(function () {
  'use strict';

  function number(value) {
    return new Intl.NumberFormat().format(value || 0);
  }

  async function load(root) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) return;
    var refresh = root.querySelector('[data-campaign-summary-refresh]');
    var status = root.querySelector('[data-campaign-summary-status]');
    var cards = root.querySelector('[data-campaign-summary-cards]');
    refresh.disabled = true;
    status.textContent = 'Loading campaign totals…';
    try {
      var token = await user.getIdToken();
      var response = await fetch(root.getAttribute('data-summary-endpoint'), {
        method: 'POST',
        headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
        body: '{}'
      });
      var data = await response.json();
      if (!response.ok) throw new Error(data.error || 'Campaign totals could not be loaded.');
      cards.replaceChildren(
        card('Email', data.email.available ? number(data.email.delivered) + ' delivered of ' + number(data.email.sent) + ' sent' : 'Unavailable',
          data.email.available ? number(data.email.awaitingDelivery) + ' awaiting delivery · ' + number(data.email.undeliverable) + ' undeliverable (' + number(data.email.bounced) + ' bounced, ' + number(data.email.suppressed) + ' suppressed, ' + number(data.email.providerFailed) + ' failed) · ' + number(data.email.delayed) + ' delayed · ' + number(data.email.opened) + ' opened · ' + number(data.email.clicked) + ' clicked' + (data.email.message ? '. ' + data.email.message : '') : data.email.message),
        card('Instagram', data.instagram.available ? number(data.instagram.published) + ' published' : 'Unavailable',
          data.instagram.available ? number(data.instagram.views) + ' views · ' + number(data.instagram.reach) + ' reach · ' + number(data.instagram.totalInteractions) + ' interactions' + (data.instagram.message ? '. ' + data.instagram.message : '') : data.instagram.message),
        card('Facebook', 'Manual outreach', data.facebook.message)
      );
      status.textContent = 'Campaign totals updated.';
    } catch (error) {
      status.textContent = error.message;
    }
    refresh.disabled = false;
  }

  function card(title, headline, detail) {
    var item = document.createElement('article');
    item.className = 'campaign-summary-card';
    var heading = document.createElement('h3');
    heading.textContent = title;
    var value = document.createElement('p');
    value.className = 'campaign-summary-card__value';
    value.textContent = headline;
    var copy = document.createElement('p');
    copy.textContent = detail || '';
    item.append(heading, value, copy);
    return item;
  }

  document.querySelectorAll('[data-campaign-summary]').forEach(function (root) {
    root.querySelector('[data-campaign-summary-refresh]').addEventListener('click', function () { load(root); });
    var container = root.closest('[data-admin-container]');
    if (container) {
      new MutationObserver(function () {
        if (!root.closest('[data-admin-content]').hidden && !root.dataset.loaded) {
          root.dataset.loaded = 'true';
          load(root);
        }
      }).observe(container, { attributes: true, subtree: true, attributeFilter: ['hidden'] });
    }
  });
}());
