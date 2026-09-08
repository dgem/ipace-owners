(function () {
  'use strict';

  var root = document.querySelector('[data-marketing-message]');
  if (!root) return;

  var form = root.querySelector('[data-marketing-message-form]');
  var sendForm = root.querySelector('[data-marketing-message-send]');
  var templateSelect = root.querySelector('[data-marketing-message-template]');
  var templateID = root.querySelector('[data-marketing-message-template-id]');
  var campaignID = root.querySelector('[data-marketing-message-campaign-id]');
  var name = root.querySelector('[data-marketing-message-name]');
  var subject = root.querySelector('[data-marketing-message-subject]');
  var markdown = root.querySelector('[data-marketing-message-markdown]');
  var confirm = root.querySelector('[data-marketing-message-confirm]');
  var status = root.querySelector('[data-marketing-message-status]');
  var preview = root.querySelector('[data-marketing-message-preview-panel]');
  var sendButton = root.querySelector('[data-marketing-message-send-button]');
  var deliveries = root.querySelector('[data-marketing-message-deliveries]');
  var deliverySummary = root.querySelector('[data-marketing-message-delivery-summary]');
  var deliveryList = root.querySelector('[data-marketing-message-delivery-list]');
  var current;
  var templates;
  var loadingTemplate = false;
  var sending = false;

  function newCampaignID() {
    if (window.crypto && window.crypto.randomUUID) return 'marketing_' + window.crypto.randomUUID().replace(/-/g, '');
    return 'marketing_' + Date.now() + '_' + Math.random().toString(36).slice(2);
  }

  campaignID.value = newCampaignID();

  function authHeaders(base) {
    return window.ipaceAuthHeaders ? window.ipaceAuthHeaders(base) : base;
  }

  function request(path, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) return Promise.reject(new Error('Sign in as an administrator first.'));
    return user.getIdToken().then(function (token) {
      return fetch(path, {
        method: 'POST',
        headers: authHeaders({'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json'}),
        body: JSON.stringify(body || {})
      });
    }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok) {
          var error = new Error(data.error || 'Request failed.');
          error.status = response.status;
          error.data = data;
          throw error;
        }
        return data;
      });
    });
  }

  function payload() {
    return {
      campaignId: campaignID.value,
      templateId: templateID.value,
      name: name.value,
      subject: subject.value,
      markdown: markdown.value,
      expectedEligible: current ? current.eligible : 0,
      confirmation: confirm.value
    };
  }

  function loadDeliveries() {
    if (!campaignID.value) return;
    request('/api/admin/marketing-message-deliveries', {campaignId: campaignID.value}).then(function (data) {
      var items = data.deliveries || [];
      var sent = items.filter(function (item) { return item.status === 'sent'; }).length;
      var held = items.length - sent;
      deliveries.hidden = false;
      deliverySummary.textContent = 'Campaign history: ' + sent + ' recipient(s) were sent previously. The date below is the original send time, not the time you opened this page.' + (held ? ' ' + held + ' recipient(s) are held for review and will not be retried automatically.' : '');
      while (deliveryList.firstChild) deliveryList.removeChild(deliveryList.firstChild);
      items.forEach(function (item) {
        var entry = document.createElement('li');
        entry.textContent = (item.maskedRecipient || 'Unknown recipient') + ' — ' + deliveryStatusLabel(item) + (item.resendId ? ' (' + item.resendId + ')' : '');
        deliveryList.appendChild(entry);
      });
    }).catch(function (error) {
      status.textContent = error.message;
    });
  }

  function deliveryStatusLabel(item) {
    if (item.status === 'sent') return 'already sent on ' + deliveryTimestamp(item.sentAt);
    if (item.status === 'failed') return 'delivery problem — held for review';
    if (item.status === 'attempting') return 'delivery status uncertain — held for review';
    return 'held for review';
  }

  function deliveryTimestamp(value) {
    var timestamp = new Date(value);
    if (isNaN(timestamp.getTime())) return 'an earlier date';
    return timestamp.toLocaleString('en-GB', {dateStyle: 'medium', timeStyle: 'short'});
  }

  function invalidate() {
    current = null;
    campaignID.value = newCampaignID();
    confirm.value = '';
    preview.hidden = true;
    sendForm.hidden = true;
    deliveries.hidden = true;
  }

  function editedMessage() {
    if (!loadingTemplate && templateID.value) {
      templateID.value = '';
      templateSelect.value = '';
    }
    invalidate();
  }

  function loadTemplates() {
    if (templates) return Promise.resolve(templates);
    return request('/api/admin/marketing-message-templates').then(function (data) {
      templates = data.templates || [];
      return templates;
    });
  }

  function applyTemplate(id) {
    if (!id) {
      templateID.value = '';
      invalidate();
      return;
    }
    status.textContent = 'Loading prepared campaign…';
    loadTemplates().then(function (items) {
      var selected = items.filter(function (item) { return item.id === id; })[0];
      if (!selected) throw new Error('That prepared campaign is unavailable.');
      loadingTemplate = true;
      templateID.value = selected.id;
      name.value = selected.name;
      subject.value = selected.subject;
      markdown.value = selected.markdown;
      loadingTemplate = false;
      invalidate();
      status.textContent = selected.description + ' Review the copy, then preview the consented audience.';
    }).catch(function (error) {
      loadingTemplate = false;
      templateSelect.value = '';
      templateID.value = '';
      status.textContent = error.message;
    });
  }

  [name, subject, markdown].forEach(function (field) {
    field.addEventListener('input', editedMessage);
  });
  templateSelect.addEventListener('change', function () { applyTemplate(templateSelect.value); });

  form.addEventListener('submit', function (event) {
    event.preventDefault();
    status.textContent = 'Calculating the current consented audience…';
    request('/api/admin/marketing-message-preview', payload()).then(function (data) {
      current = data;
      campaignID.value = data.campaignId || campaignID.value;
      preview.hidden = false;
      sendForm.hidden = false;
      root.querySelector('[data-marketing-message-audience]').textContent = data.eligible + ' members currently consent to group communications.';
      root.querySelector('[data-marketing-message-subject-preview]').textContent = data.subject;
      root.querySelector('[data-marketing-message-html]').srcdoc = data.html;
      root.querySelector('[data-marketing-message-text]').textContent = data.text;
      root.querySelector('[data-marketing-message-confirm-hint]').textContent = 'Type “' + data.confirmation + '” exactly to send the next batch.';
      status.textContent = data.notice;
    }).catch(function (error) {
      status.textContent = error.message;
    });
  });

  sendForm.addEventListener('submit', function (event) {
    event.preventDefault();
    if (!current || sending) return;
    sending = true;
    sendButton.disabled = true;
    status.textContent = 'Sending the next batch of up to 100 consented members…';
    request('/api/admin/marketing-message-send', payload()).then(function (data) {
      current.eligible = data.eligible;
      campaignID.value = data.campaignId;
      confirm.value = '';
      status.textContent = 'Batch complete: ' + data.batchSent + ' new email(s) sent' + (data.batchFailed ? ', and ' + data.batchFailed + ' held for review' : '') + '. ' + data.remaining + ' recipient(s) remain.';
      root.querySelector('[data-marketing-message-audience]').textContent = 'Campaign progress: ' + data.sent + ' already sent, ' + data.failed + ' held for review, and ' + data.remaining + ' remaining out of ' + data.eligible + ' consented members.';
      loadDeliveries();
      if (data.remaining === 0) {
        sendForm.hidden = true;
      }
    }).catch(function (error) {
      if (error.status === 409 && error.data && error.data.eligible) {
        current.eligible = error.data.eligible;
        confirm.value = '';
        root.querySelector('[data-marketing-message-audience]').textContent = error.data.eligible + ' members currently consent to group communications.';
        root.querySelector('[data-marketing-message-confirm-hint]').textContent = 'The audience changed. Type “' + error.data.confirmation + '” to confirm the updated count.';
      }
      if (error.message.indexOf('earlier provider failure without a recipient ledger entry') !== -1) {
        loadDeliveries();
      }
      status.textContent = error.message;
    }).finally(function () {
      sending = false;
      sendButton.disabled = false;
    });
  });
}());
