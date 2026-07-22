(function () {
  'use strict';

  async function request(path, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) throw new Error('Sign in as an administrator first.');
    var token = await user.getIdToken();
    var response = await fetch(path, {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: JSON.stringify(body || {})
    });
    var data = await response.json();
    if (!response.ok) throw new Error(data.error || 'The campaign request failed.');
    return data;
  }

  function initialise(root) {
    var previewButton = root.querySelector('[data-campaign-preview]');
    var summary = root.querySelector('[data-campaign-summary]');
    var form = root.querySelector('[data-campaign-send]');
    var confirmInput = root.querySelector('[data-campaign-confirm]');
    var confirmHint = root.querySelector('[data-campaign-confirm-hint]');
    var sendButton = root.querySelector('[data-campaign-send-button]');
    var emailPreview = root.querySelector('[data-campaign-email-preview]');
    var emailSubject = root.querySelector('[data-campaign-email-subject]');
    var emailText = root.querySelector('[data-campaign-email-text]');
    var sharePreview = root.querySelector('[data-campaign-share-preview]');
    var status = root.querySelector('[data-campaign-status]');
    var current = null;

    function renderShares(shares) {
      if (!sharePreview) return;
      sharePreview.textContent = '';
      (shares || []).forEach(function (share) {
        var item = document.createElement('span');
        item.className = 'email-preview__share';
        var mark = document.createElement('span');
        mark.className = 'email-preview__share-mark';
        mark.textContent = share.mark;
        item.appendChild(mark);
        item.appendChild(document.createTextNode(' ' + share.label));
        sharePreview.appendChild(item);
      });
      sharePreview.hidden = !shares || shares.length === 0;
    }

    function render(data) {
      current = data;
      summary.textContent = '';
      var p = document.createElement('p');
      p.textContent = data.eligible + ' eligible; ' + data.sent + ' already sent in campaign ' + data.campaignId + '.';
      summary.appendChild(p);
      var remaining = data.remaining;
      emailSubject.textContent = data.emailPreview.subject;
      emailText.textContent = data.emailPreview.text;
      renderShares(data.emailPreview.shares);
      emailPreview.hidden = false;
      confirmInput.disabled = remaining < 1;
      sendButton.disabled = remaining < 1;
      confirmHint.textContent = remaining > 0 ? 'Type “SEND ' + data.eligible + '” exactly. ' + remaining + ' remain.' : 'Everyone in this campaign has already been sent an email.';
      sendButton.textContent = remaining > 10 ? 'Send next 10 emails' : 'Send final ' + remaining + ' email' + (remaining === 1 ? '' : 's');
    }

    previewButton.addEventListener('click', async function () {
      previewButton.disabled = true;
      status.textContent = 'Calculating the current audience…';
      try {
        render(await request(root.getAttribute('data-preview-endpoint')));
        status.textContent = 'Preview complete. No email was sent.';
      } catch (error) {
        status.textContent = error.message;
      }
      previewButton.disabled = false;
    });

    form.addEventListener('submit', async function (event) {
      event.preventDefault();
      if (!current) return;
      sendButton.disabled = true;
      status.textContent = 'Sending a bounded batch…';
      try {
        var data = await request(root.getAttribute('data-send-endpoint'), {
          campaignId: current.campaignId,
          expectedEligible: current.eligible,
          confirmation: confirmInput.value
        });
        confirmInput.value = '';
        render(data);
        status.textContent = data.batchSent + ' accepted by the email provider in this batch; ' + data.remaining + ' remain.';
      } catch (error) {
        status.textContent = error.message;
      }
      sendButton.disabled = !current || current.remaining < 1;
    });
  }

  document.querySelectorAll('[data-email-campaign]').forEach(initialise);
}());
