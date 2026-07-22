(function () {
  'use strict';

  var root = document.querySelector('[data-email-campaigns]');
  if (!root) return;
  var previewButton = root.querySelector('[data-campaign-preview]');
  var summary = root.querySelector('[data-campaign-summary]');
  var form = root.querySelector('[data-campaign-send]');
  var confirmInput = root.querySelector('[data-campaign-confirm]');
  var confirmHint = root.querySelector('[data-campaign-confirm-hint]');
  var sendButton = root.querySelector('[data-campaign-send-button]');
  var emailPreview = root.querySelector('[data-campaign-email-preview]');
  var emailSubject = root.querySelector('[data-campaign-email-subject]');
  var emailText = root.querySelector('[data-campaign-email-text]');
  var status = root.querySelector('[data-campaign-status]');
  var current = null;

  async function request(path, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) throw new Error('Sign in as an administrator first.');
    var token = await user.getIdToken();
    var response = await fetch(path, { method: 'POST', headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }, body: JSON.stringify(body || {}) });
    var data = await response.json();
    if (!response.ok) throw new Error(data.error || 'The campaign request failed.');
    return data;
  }

  function render(data) {
    current = data;
    summary.textContent = '';
    var p = document.createElement('p');
    p.textContent = data.eligible + ' eligible; ' + data.registered + ' already registered; ' + data.sent + ' already sent in campaign ' + data.campaignId + '.';
    summary.appendChild(p);
    var remaining = data.eligible - data.sent;
    emailSubject.textContent = data.emailPreview.subject;
    emailText.textContent = data.emailPreview.text;
    emailPreview.hidden = false;
    confirmInput.disabled = remaining < 1;
    sendButton.disabled = remaining < 1;
    confirmHint.textContent = remaining > 0 ? 'Type “SEND ' + data.eligible + '” exactly. ' + remaining + ' remain.' : 'Everyone in this campaign has already been sent an email.';
    sendButton.textContent = remaining > 10 ? 'Send next 10 reminder emails' : 'Send final ' + remaining + ' reminder email' + (remaining === 1 ? '' : 's');
  }

  async function preview() {
    previewButton.disabled = true; status.textContent = 'Calculating the current audience…';
    try { render(await request('/api/admin/reengagement-preview')); status.textContent = 'Preview complete. No email was sent.'; }
    catch (error) { status.textContent = error.message; }
    previewButton.disabled = false;
  }

  previewButton.addEventListener('click', preview);
  form.addEventListener('submit', async function (event) {
    event.preventDefault(); if (!current) return;
    sendButton.disabled = true; status.textContent = 'Sending a bounded batch…';
    try {
      var data = await request('/api/admin/reengagement-send', { campaignId: current.campaignId, expectedEligible: current.eligible, confirmation: confirmInput.value });
      confirmInput.value = ''; render(data); status.textContent = data.batchSent + ' accepted by the email provider in this batch; ' + data.remaining + ' remain.';
    } catch (error) { status.textContent = error.message; }
    sendButton.disabled = !current || current.remaining < 1;
  });
}());
