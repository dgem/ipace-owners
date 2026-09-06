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
  var current;
  var templates;
  var loadingTemplate = false;

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
        if (!response.ok) throw new Error(data.error || 'Request failed.');
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

  function invalidate() {
    current = null;
    campaignID.value = newCampaignID();
    confirm.value = '';
    preview.hidden = true;
    sendForm.hidden = true;
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
    if (!current) return;
    status.textContent = 'Sending the next batch of up to 100 consented members…';
    request('/api/admin/marketing-message-send', payload()).then(function (data) {
      current.eligible = data.eligible;
      campaignID.value = data.campaignId;
      confirm.value = '';
      status.textContent = data.message;
      root.querySelector('[data-marketing-message-audience]').textContent = data.sent + ' of ' + data.eligible + ' consented members emailed. ' + data.remaining + ' remain.';
      if (data.remaining === 0) {
        sendForm.hidden = true;
      }
    }).catch(function (error) {
      status.textContent = error.message;
    });
  });
}());
