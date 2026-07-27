(function () {
  'use strict';

  async function request(path, body) {
    var user = window.firebase && window.firebase.auth().currentUser;
    if (!user) throw new Error('Sign in as an administrator first.');
    var token = await user.getIdToken();
    var response = await fetch(path, {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
    var data = await response.json();
    if (!response.ok) throw new Error(data.error || 'The Instagram campaign request failed.');
    return data;
  }

  function initialise(root) {
    var generateForm = root.querySelector('[data-veo-generate]');
    var generatePrompt = root.querySelector('[data-veo-prompt]');
    var generateConfirm = root.querySelector('[data-veo-confirm]');
    var generateButton = root.querySelector('[data-veo-generate-button]');
    var generateStatus = root.querySelector('[data-veo-status]');
    var generateResult = root.querySelector('[data-veo-result]');
    var generateJobID = root.querySelector('[data-veo-job-id]');
    var generateVideo = root.querySelector('[data-veo-video]');
    var draftForm = root.querySelector('[data-instagram-draft]');
    var publishForm = root.querySelector('[data-instagram-publish]');
    var name = root.querySelector('[data-instagram-name]');
    var mediaPath = root.querySelector('[data-instagram-media-path]');
    var caption = root.querySelector('[data-instagram-caption]');
    var mediaReviewed = root.querySelector('[data-instagram-media-reviewed]');
    var count = root.querySelector('[data-instagram-caption-count]');
    var previewButton = root.querySelector('[data-instagram-preview-button]');
    var preview = root.querySelector('[data-instagram-preview]');
    var previewVideo = root.querySelector('[data-instagram-video]');
    var previewCaption = root.querySelector('[data-instagram-caption-preview]');
    var campaignID = root.querySelector('[data-instagram-campaign-id]');
    var confirm = root.querySelector('[data-instagram-confirm]');
    var confirmHint = root.querySelector('[data-instagram-confirm-hint]');
    var publishButton = root.querySelector('[data-instagram-publish-button]');
    var status = root.querySelector('[data-instagram-status]');
    var historyList = root.querySelector('[data-instagram-history-list]');
    var historyStatus = root.querySelector('[data-instagram-history-status]');
    var historyRefresh = root.querySelector('[data-instagram-history-refresh]');
    var newButton = root.querySelector('[data-instagram-new]');
    var workspaceTitle = root.querySelector('[data-instagram-workspace-title]');
    var current = null;
    var savedCampaignID = '';
    var sourceCampaignID = '';
    var generationTimer = null;
    var generationStorageKey = 'ipace-instagram-veo-job';

    function draft() {
      return {
        campaignId: savedCampaignID,
        sourceCampaignId: sourceCampaignID,
        name: name.value,
        mediaPath: mediaPath.value,
        caption: caption.value,
        mediaReviewed: mediaReviewed.checked
      };
    }
    function invalidate() {
      current = null;
      preview.hidden = true;
      confirm.value = '';
      confirm.disabled = true;
      publishButton.disabled = true;
      confirmHint.textContent = 'Validate the draft first.';
    }
    function updateCount() { count.textContent = caption.value.length; }
    caption.addEventListener('input', function () { updateCount(); invalidate(); });
    name.addEventListener('input', invalidate);
    mediaPath.addEventListener('input', invalidate);
    mediaReviewed.addEventListener('change', invalidate);
    updateCount();

    function addStat(container, label, value) {
      var item = document.createElement('span');
      item.className = 'email-campaign-history__stat';
      item.textContent = label + ': ' + value;
      container.appendChild(item);
    }

    function campaignDate(value) {
      if (!value) return '';
      var date = new Date(value);
      return Number.isNaN(date.getTime()) ? '' : date.toLocaleString();
    }

    function openCampaign(campaign, clone) {
      savedCampaignID = clone ? '' : campaign.campaignId;
      sourceCampaignID = clone ? campaign.campaignId : (campaign.sourceCampaignId || '');
      name.value = clone ? campaign.name + ' repost' : campaign.name;
      mediaPath.value = campaign.mediaPath;
      caption.value = campaign.caption;
      mediaReviewed.checked = false;
      workspaceTitle.textContent = clone ? 'Edit and repost “' + campaign.name + '”' : 'Edit “' + campaign.name + '”';
      updateCount();
      invalidate();
      status.textContent = clone ? 'A new campaign will be saved; the original remains unchanged.' : 'Draft loaded. Preview it again before publishing.';
      document.getElementById('instagram-workspace').scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    function renderHistory(history) {
      historyList.replaceChildren();
      var campaigns = history.campaigns || [];
      if (!campaigns.length) {
        var empty = document.createElement('p');
        empty.textContent = 'No Instagram campaigns have been saved yet.';
        historyList.appendChild(empty);
      }
      campaigns.forEach(function (campaign) {
        var item = document.createElement('article');
        item.className = 'email-campaign-history__item';
        var title = document.createElement('h3');
        title.textContent = campaign.name || 'Instagram campaign';
        var meta = document.createElement('p');
        meta.className = 'form-hint';
        meta.textContent = campaign.status + (campaign.updatedAt ? ' · updated ' + campaignDate(campaign.updatedAt) : '');
        var stats = document.createElement('div');
        stats.className = 'email-campaign-history__stats';
        if (campaign.insights && campaign.insights.available) {
          addStat(stats, 'Views', campaign.insights.views);
          addStat(stats, 'Reach', campaign.insights.reach);
          addStat(stats, 'Interactions', campaign.insights.totalInteractions);
        } else if (campaign.status === 'published') {
          addStat(stats, 'Insights', history.insightsConfigured ? 'not available yet' : 'not configured');
        }
        var action = document.createElement('button');
        action.type = 'button';
        action.className = 'btn btn--secondary';
        action.textContent = campaign.status === 'draft' ? 'Edit draft' : 'Edit and repost';
        action.addEventListener('click', function () { openCampaign(campaign, campaign.status !== 'draft'); });
        item.append(title, meta, stats, action);
        historyList.appendChild(item);
      });
      historyStatus.textContent = campaigns.length + (campaigns.length === 1 ? ' campaign loaded.' : ' campaigns loaded.');
    }

    async function loadHistory() {
      historyRefresh.disabled = true;
      historyStatus.textContent = 'Loading Instagram campaign history…';
      try {
        renderHistory(await request(root.getAttribute('data-history-endpoint'), {}));
      } catch (error) {
        historyStatus.textContent = error.message;
      }
      historyRefresh.disabled = false;
    }

    function createNew() {
      savedCampaignID = '';
      sourceCampaignID = '';
      current = null;
      name.value = '';
      mediaPath.value = '';
      mediaReviewed.checked = false;
      workspaceTitle.textContent = 'Create an Instagram campaign';
      invalidate();
      status.textContent = 'New campaign ready.';
      document.getElementById('instagram-workspace').scrollIntoView({ behavior: 'smooth', block: 'start' });
    }

    historyRefresh.addEventListener('click', loadHistory);
    newButton.addEventListener('click', createNew);

    function requestID() {
      if (window.crypto && typeof window.crypto.randomUUID === 'function') return window.crypto.randomUUID();
      return String(Date.now()) + '-' + Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2);
    }

    function rememberJob(jobID) {
      try { window.localStorage.setItem(generationStorageKey, jobID); } catch { /* storage is optional */ }
    }

    function forgetJob() {
      try { window.localStorage.removeItem(generationStorageKey); } catch { /* storage is optional */ }
    }

    function renderGeneration(result) {
      generateJobID.textContent = result.jobId;
      generateResult.hidden = false;
      generateStatus.textContent = result.message;
      rememberJob(result.jobId);
      if (result.status === 'completed' && result.mediaPath) {
        if (generationTimer) window.clearTimeout(generationTimer);
        generationTimer = null;
        generateVideo.src = result.mediaPath;
        generateVideo.hidden = false;
        mediaPath.value = result.mediaPath;
        mediaReviewed.checked = false;
        invalidate();
        generateButton.disabled = false;
        forgetJob();
        return;
      }
      if (result.status === 'failed') {
        generateButton.disabled = false;
        forgetJob();
        return;
      }
      generationTimer = window.setTimeout(function () { pollGeneration(result.jobId); }, 15000);
    }

    async function pollGeneration(jobID) {
      try {
        renderGeneration(await request(root.getAttribute('data-generation-status-endpoint'), { jobId: jobID }));
      } catch (error) {
        generateStatus.textContent = error.message + ' Retrying shortly…';
        generationTimer = window.setTimeout(function () { pollGeneration(jobID); }, 15000);
      }
    }

    generateForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      generateButton.disabled = true;
      generateStatus.textContent = 'Submitting the billable Veo generation job…';
      try {
        var result = await request(root.getAttribute('data-generate-endpoint'), {
          requestId: requestID(),
          prompt: generatePrompt.value,
          confirmation: generateConfirm.value
        });
        generateConfirm.value = '';
        renderGeneration(result);
      } catch (error) {
        generateStatus.textContent = error.message;
        generateButton.disabled = false;
      }
    });

    try {
      var rememberedJob = window.localStorage.getItem(generationStorageKey);
      if (rememberedJob) {
        generateButton.disabled = true;
        generateStatus.textContent = 'Restoring the previous Veo generation job…';
        pollGeneration(rememberedJob);
      }
    } catch { /* storage is optional */ }

    draftForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      previewButton.disabled = true;
      status.textContent = 'Validating the draft…';
      try {
        current = await request(root.getAttribute('data-preview-endpoint'), draft());
        savedCampaignID = current.campaignId;
        sourceCampaignID = current.sourceCampaignId || sourceCampaignID;
        previewVideo.src = current.mediaPath;
        previewCaption.textContent = current.caption;
        campaignID.textContent = current.campaignId;
        preview.hidden = false;
        confirm.disabled = !current.configured;
        publishButton.disabled = !current.configured;
        confirmHint.textContent = current.configured ? 'Type “' + current.confirmation + '” exactly.' : 'Instagram publishing credentials must be configured before this draft can be published.';
        status.textContent = current.configured ? 'Preview complete. Nothing has been published.' : 'Preview complete, but Instagram publishing is not configured on this environment.';
        loadHistory();
      } catch (error) {
        invalidate();
        status.textContent = error.message;
      }
      previewButton.disabled = false;
    });

    publishForm.addEventListener('submit', async function (event) {
      event.preventDefault();
      if (!current) return;
      publishButton.disabled = true;
      status.textContent = 'Uploading, processing and publishing the Reel…';
      try {
        var result = await request(root.getAttribute('data-publish-endpoint'), {
          campaignId: current.campaignId,
          name: current.name,
          mediaPath: current.mediaPath,
          caption: current.caption,
          mediaReviewed: true,
          confirmation: confirm.value
        });
        confirm.value = '';
        confirm.disabled = true;
        status.textContent = 'Published successfully as Instagram media ' + result.mediaId + '.';
        loadHistory();
      } catch (error) {
        status.textContent = error.message;
        publishButton.disabled = !current.configured;
      }
    });

    loadHistory();
  }

  document.querySelectorAll('[data-instagram-campaign]').forEach(initialise);
}());
