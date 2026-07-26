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

  function formatDate(value) {
    if (!value) return '—';
    var date = new Date(value);
    return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString();
  }

  var selectCampaignTab = function () {};

  function initialiseCampaignTabs() {
    var tablist = document.querySelector('[data-campaign-tabs]');
    if (!tablist) return;
    var tabs = Array.prototype.slice.call(tablist.querySelectorAll('[data-campaign-tab]'));
    var panels = Array.prototype.slice.call(document.querySelectorAll('[data-campaign-panel]'));

    function activate(name, moveFocus) {
      var selectedTab = null;
      tabs.forEach(function (tab) {
        var selected = tab.getAttribute('data-campaign-tab') === name;
        tab.setAttribute('aria-selected', selected ? 'true' : 'false');
        tab.tabIndex = selected ? 0 : -1;
        if (selected) selectedTab = tab;
      });
      panels.forEach(function (panel) {
        panel.hidden = panel.getAttribute('data-campaign-panel') !== name;
      });
      if (moveFocus && selectedTab) selectedTab.focus();
    }

    selectCampaignTab = activate;
    tablist.addEventListener('click', function (event) {
      var tab = event.target.closest('[data-campaign-tab]');
      if (tab) activate(tab.getAttribute('data-campaign-tab'));
    });
    tablist.addEventListener('keydown', function (event) {
      var currentIndex = tabs.indexOf(event.target.closest('[data-campaign-tab]'));
      if (currentIndex < 0) return;
      var nextIndex = currentIndex;
      if (event.key === 'ArrowRight') nextIndex = (currentIndex + 1) % tabs.length;
      if (event.key === 'ArrowLeft') nextIndex = (currentIndex - 1 + tabs.length) % tabs.length;
      if (event.key === 'Home') nextIndex = 0;
      if (event.key === 'End') nextIndex = tabs.length - 1;
      if (nextIndex === currentIndex && event.key !== 'Home' && event.key !== 'End') return;
      event.preventDefault();
      activate(tabs[nextIndex].getAttribute('data-campaign-tab'), true);
    });
    document.querySelectorAll('[data-campaign-open-tab]').forEach(function (control) {
      control.addEventListener('click', function () {
        activate(control.getAttribute('data-campaign-open-tab'));
      });
    });
    var selected = tabs.find(function (tab) { return tab.getAttribute('aria-selected') === 'true'; });
    activate(selected ? selected.getAttribute('data-campaign-tab') : tabs[0].getAttribute('data-campaign-tab'));
  }

  function initialiseCustomCampaign() {
    var root = document.querySelector('[data-custom-email-campaign]');
    var historyRoot = document.querySelector('[data-email-campaign-history]');
    if (!root || !historyRoot) return;

    var form = root.querySelector('[data-custom-campaign-form]');
    var nameInput = root.querySelector('[data-custom-campaign-name]');
    var subjectInput = root.querySelector('[data-custom-campaign-subject]');
    var markdownInput = root.querySelector('[data-custom-campaign-markdown]');
    var placeholderList = root.querySelector('[data-custom-campaign-placeholders]');
    var newButton = root.querySelector('[data-custom-campaign-new]');
    var preview = root.querySelector('[data-custom-campaign-email-preview]');
    var previewSubject = root.querySelector('[data-custom-campaign-email-subject]');
    var previewHTML = root.querySelector('[data-custom-campaign-email-html]');
    var previewText = root.querySelector('[data-custom-campaign-email-text]');
    var summary = root.querySelector('[data-custom-campaign-summary]');
    var sendForm = root.querySelector('[data-custom-campaign-send]');
    var confirmInput = root.querySelector('[data-custom-campaign-confirm]');
    var confirmHint = root.querySelector('[data-custom-campaign-confirm-hint]');
    var sendButton = root.querySelector('[data-custom-campaign-send-button]');
    var status = root.querySelector('[data-custom-campaign-status]');
    var historyList = historyRoot.querySelector('[data-campaign-history-list]');
    var historyStatus = historyRoot.querySelector('[data-campaign-history-status]');
    var historyRefresh = historyRoot.querySelector('[data-campaign-history-refresh]');
    var draftId = '';
    var sourceId = '';
    var current = null;
    var historyLoaded = false;

    function setDeliveryState(data) {
      current = data;
      summary.textContent = data.eligible + ' eligible; ' + data.sent + ' sent; ' + data.failed + ' failed attempts; ' + data.remaining + ' remaining.';
      previewSubject.textContent = data.emailPreview.subject;
      previewHTML.srcdoc = data.emailPreview.html;
      previewText.textContent = data.emailPreview.text;
      preview.hidden = false;
      confirmInput.disabled = data.remaining < 1;
      sendButton.disabled = data.remaining < 1;
      confirmHint.textContent = data.remaining > 0 ? 'Type “SEND ' + data.eligible + '” exactly.' : 'This campaign run is complete.';
      sendButton.textContent = data.remaining > 10 ? 'Send next 10 emails' : 'Send final ' + data.remaining + ' email' + (data.remaining === 1 ? '' : 's');
    }

    function campaignChanged() {
      if (current && current.sent > 0) {
        sourceId = draftId;
        draftId = '';
        status.textContent = 'Your edit will be saved as a new campaign run after preview.';
      } else {
        status.textContent = 'Content changed. Save and preview again before sending.';
      }
      current = null;
      confirmInput.disabled = true;
      sendButton.disabled = true;
    }

    [nameInput, subjectInput, markdownInput].forEach(function (input) {
      input.addEventListener('input', campaignChanged);
    });

    function clearEditor() {
      draftId = '';
      sourceId = '';
      current = null;
      form.reset();
      summary.textContent = '';
      preview.hidden = true;
      confirmInput.value = '';
      confirmInput.disabled = true;
      sendButton.disabled = true;
      status.textContent = 'New campaign ready to write.';
      nameInput.focus();
    }

    newButton.addEventListener('click', clearEditor);

    function insertPlaceholder(name) {
      var token = '{{' + name + '}}';
      var start = markdownInput.selectionStart;
      var end = markdownInput.selectionEnd;
      markdownInput.setRangeText(token, start, end, 'end');
      markdownInput.focus();
      campaignChanged();
    }

    function renderPlaceholders(names) {
      placeholderList.textContent = '';
      names.forEach(function (name) {
        var button = document.createElement('button');
        button.type = 'button';
        button.className = 'email-campaign-placeholder';
        button.textContent = '{{' + name + '}}';
        button.setAttribute('data-custom-placeholder', name);
        placeholderList.appendChild(button);
      });
    }

    placeholderList.addEventListener('click', function (event) {
      var button = event.target.closest('[data-custom-placeholder]');
      if (button) insertPlaceholder(button.getAttribute('data-custom-placeholder'));
    });

    function addStat(parent, label, value) {
      var item = document.createElement('span');
      item.className = 'email-campaign-history__stat';
      item.textContent = label + ': ' + value;
      parent.appendChild(item);
    }

    function loadForRerun(campaign) {
      selectCampaignTab('freeform');
      draftId = '';
      sourceId = campaign.campaignId;
      current = null;
      nameInput.value = campaign.name + ' — rerun';
      subjectInput.value = campaign.subject || '';
      markdownInput.value = campaign.markdown || '';
      preview.hidden = true;
      summary.textContent = '';
      confirmInput.disabled = true;
      sendButton.disabled = true;
      status.textContent = 'Loaded from ' + campaign.campaignId + '. Edit it, then save and preview the new run.';
      root.scrollIntoView({ behavior: 'smooth', block: 'start' });
      nameInput.focus();
    }

    function loadExistingRun(campaign) {
      selectCampaignTab('freeform');
      draftId = campaign.campaignId;
      sourceId = campaign.sourceCampaignId || '';
      current = { sent: campaign.sent || 0 };
      nameInput.value = campaign.name || '';
      subjectInput.value = campaign.subject || '';
      markdownInput.value = campaign.markdown || '';
      preview.hidden = true;
      summary.textContent = '';
      confirmInput.disabled = true;
      sendButton.disabled = true;
      status.textContent = campaign.sent > 0 ? 'Campaign loaded. Preview the unchanged content to continue its remaining deliveries.' : 'Draft loaded. Edit and preview it when ready.';
      root.scrollIntoView({ behavior: 'smooth', block: 'start' });
      nameInput.focus();
    }

    function renderHistory(data) {
      renderPlaceholders(data.placeholders || []);
      historyList.textContent = '';
      if (!data.campaigns || data.campaigns.length === 0) {
        historyList.textContent = 'No campaign runs have been recorded yet.';
        return;
      }
      data.campaigns.forEach(function (campaign) {
        var article = document.createElement('article');
        article.className = 'email-campaign-history__item';
        var heading = document.createElement('h3');
        heading.textContent = campaign.name || campaign.campaignId;
        var meta = document.createElement('p');
        meta.className = 'form-hint';
        meta.textContent = campaign.campaignId + ' · ' + (campaign.status || 'recorded') + ' · updated ' + formatDate(campaign.updatedAt || campaign.lastSentAt);
        var stats = document.createElement('div');
        stats.className = 'email-campaign-history__stats';
        addStat(stats, 'Eligible', campaign.eligible || 0);
        addStat(stats, 'Sent', campaign.sent || 0);
        addStat(stats, 'Remaining', campaign.remaining || 0);
        addStat(stats, 'Failed attempts', campaign.failed || 0);
        addStat(stats, 'Batches', campaign.batchCount || 0);
        var actions = document.createElement('div');
        actions.className = 'cluster';
        var canRerun = campaign.kind === 'custom-member' || campaign.kind === 'member-referral';
        if (campaign.kind === 'custom-member' && campaign.remaining > 0) {
          var continueButton = document.createElement('button');
          continueButton.type = 'button';
          continueButton.className = 'btn btn--primary btn--sm';
          continueButton.textContent = campaign.sent > 0 ? 'Continue sending' : 'Edit draft';
          continueButton.addEventListener('click', function () { loadExistingRun(campaign); });
          actions.appendChild(continueButton);
        }
        var rerunButton = document.createElement('button');
        rerunButton.type = 'button';
        rerunButton.className = 'btn btn--secondary btn--sm';
        rerunButton.textContent = canRerun ? 'Tweak and rerun' : 'Use registration reminder tool';
        rerunButton.disabled = canRerun && (!campaign.subject || !campaign.markdown);
        if (!canRerun) rerunButton.title = 'Registration reminders require a fresh private sign-in link and their original unverified-member audience.';
        rerunButton.addEventListener('click', function () {
          if (canRerun) {
            loadForRerun(campaign);
            return;
          }
          selectCampaignTab('registration');
          document.querySelector('#campaign-tools').scrollIntoView({ behavior: 'smooth', block: 'start' });
        });
        actions.appendChild(rerunButton);
        article.appendChild(heading);
        article.appendChild(meta);
        article.appendChild(stats);
        article.appendChild(actions);
        historyList.appendChild(article);
      });
    }

    async function loadHistory() {
      historyRefresh.disabled = true;
      historyStatus.textContent = 'Loading campaign history…';
      try {
        renderHistory(await request('/api/admin/email-campaign-history'));
        historyStatus.textContent = '';
        historyLoaded = true;
      } catch (error) {
        historyStatus.textContent = error.message;
      }
      historyRefresh.disabled = false;
    }

    historyRefresh.addEventListener('click', loadHistory);

    form.addEventListener('submit', async function (event) {
      event.preventDefault();
      var button = root.querySelector('[data-custom-campaign-preview]');
      button.disabled = true;
      status.textContent = 'Calculating the verified, consented audience and rendering a personalised preview…';
      try {
        var data = await request('/api/admin/custom-campaign-preview', {
          campaignId: draftId,
          sourceCampaignId: sourceId,
          name: nameInput.value,
          subject: subjectInput.value,
          markdown: markdownInput.value
        });
        draftId = data.campaignId;
        setDeliveryState(data);
        status.textContent = 'Draft saved and previewed. No email was sent.';
        await loadHistory();
      } catch (error) {
        status.textContent = error.message;
      }
      button.disabled = false;
    });

    sendForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      if (!current) return;
      sendButton.disabled = true;
      status.textContent = 'Sending a bounded batch…';
      try {
        var data = await request('/api/admin/custom-campaign-send', {
          campaignId: current.campaignId,
          expectedEligible: current.eligible,
          confirmation: confirmInput.value
        });
        confirmInput.value = '';
        setDeliveryState(data);
        status.textContent = data.batchSent + ' accepted by the email provider in this batch; ' + data.remaining + ' remain.';
        await loadHistory();
      } catch (error) {
        status.textContent = error.message;
      }
      sendButton.disabled = !current || current.remaining < 1;
    });

    document.addEventListener('identity:login', function () {
      if (!historyLoaded) loadHistory();
    });
    if (window.firebase && window.firebase.auth().currentUser) loadHistory();
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
    var emailHTML = root.querySelector('[data-campaign-email-html]');
    var emailText = root.querySelector('[data-campaign-email-text]');
    var status = root.querySelector('[data-campaign-status]');
    var current = null;

    function render(data) {
      current = data;
      summary.textContent = '';
      var p = document.createElement('p');
      p.textContent = data.eligible + ' eligible; ' + data.sent + ' already sent in campaign ' + data.campaignId + '.';
      summary.appendChild(p);
      var remaining = data.remaining;
      emailSubject.textContent = data.emailPreview.subject;
      emailHTML.srcdoc = data.emailPreview.html;
      emailText.textContent = data.emailPreview.text;
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

  initialiseCampaignTabs();
  document.querySelectorAll('[data-email-campaign]').forEach(initialise);
  initialiseCustomCampaign();
}());
