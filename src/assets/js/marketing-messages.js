(function () {
  'use strict';

  var root = document.querySelector('[data-marketing-message]');
  if (!root) return;

  var form = root.querySelector('[data-marketing-message-form]');
  var sendForm = root.querySelector('[data-marketing-message-send]');
  var name = root.querySelector('[data-marketing-message-name]');
  var subject = root.querySelector('[data-marketing-message-subject]');
  var markdown = root.querySelector('[data-marketing-message-markdown]');
  var confirm = root.querySelector('[data-marketing-message-confirm]');
  var status = root.querySelector('[data-marketing-message-status]');
  var preview = root.querySelector('[data-marketing-message-preview-panel]');
  var current;

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
        body: JSON.stringify(body)
      });
    }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok) throw new Error(data.error || 'Request failed.');
        return data;
      });
    });
  }

  function payload() {
    return {name: name.value, subject: subject.value, markdown: markdown.value, expectedEligible: current ? current.eligible : 0, confirmation: confirm.value};
  }

  function invalidate() {
    current = null;
    preview.hidden = true;
    sendForm.hidden = true;
  }

  [name, subject, markdown].forEach(function (field) { field.addEventListener('input', invalidate); });

  form.addEventListener('submit', function (event) {
    event.preventDefault();
    status.textContent = 'Calculating the current consented audience…';
    request('/api/admin/marketing-message-preview', payload()).then(function (data) {
      current = data;
      preview.hidden = false;
      sendForm.hidden = false;
      root.querySelector('[data-marketing-message-audience]').textContent = data.eligible + ' members currently consent to group communications.';
      root.querySelector('[data-marketing-message-subject-preview]').textContent = data.subject;
      root.querySelector('[data-marketing-message-html]').srcdoc = data.html;
      root.querySelector('[data-marketing-message-text]').textContent = data.text;
      root.querySelector('[data-marketing-message-confirm-hint]').textContent = 'Type “' + data.confirmation + '” exactly to send.';
      status.textContent = data.notice;
    }).catch(function (error) { status.textContent = error.message; });
  });

  sendForm.addEventListener('submit', function (event) {
    event.preventDefault();
    if (!current) return;
    status.textContent = 'Creating the consented audience and handing the broadcast to Resend…';
    request('/api/admin/marketing-message-send', payload()).then(function (data) {
      sendForm.hidden = true;
      status.textContent = data.message + ' Broadcast ID: ' + data.broadcastId + '.';
    }).catch(function (error) { status.textContent = error.message; });
  });
}());
