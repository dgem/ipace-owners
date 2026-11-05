(function () {
  'use strict';
  var root = document.querySelector('[data-admin-story-editor]');
  if (!root) return;
  var storyID = new URLSearchParams(window.location.search).get('id') || '';
  var impact = {};

  function esc(value) { var node = document.createElement('div'); node.textContent = value == null ? '' : String(value); return node.innerHTML; }
  function api(options) {
    return window.ipaceGetIdentityToken().then(function (token) {
      options = options || {};
      options.headers = Object.assign({ Authorization: 'Bearer ' + token }, options.headers || {});
      return fetch('/api/admin/stories' + (options.query || ''), options);
    }).then(function (response) { return response.json().then(function (data) { if (!response.ok) throw new Error(data.error || 'Request failed'); return data; }); });
  }
  function findStory(page) {
    var query = new URLSearchParams({ page: page, pageSize: 50, sort: 'updatedAt', direction: 'desc' });
    return api({ query: '?' + query }).then(function (data) {
      impact = data.impact || impact;
      var story = (data.stories || []).find(function (item) { return item.id === storyID; });
      if (story) return story;
      if (data.hasNext) return findStory(page + 1);
      throw new Error('Story not found. Return to the experience list and try again.');
    });
  }
  function load(message) {
    if (!storyID) {
      root.innerHTML = '<div class="callout callout--warning"><p>Select an experience from the review list first.</p><a class="btn btn--primary" href="/admin/member-stories/">Experience list</a></div>';
      return;
    }
    root.innerHTML = '<p>Loading experience…</p>';
    findStory(1).then(function (story) { render(story, message); }).catch(function (error) { root.innerHTML = '<div class="callout callout--warning"><p>' + esc(error.message) + '</p><a class="btn btn--secondary" href="/admin/member-stories/">Back to experiences</a></div>'; });
  }
  function render(story, message) {
    var analysis = story.analysis || {};
    var metrics = impact[story.id] || {};
    var media = (story.media || []).filter(function (item) { return item.url; }).map(function (item) {
      var preview = item.contentType.indexOf('image/') === 0 ? '<img src="' + esc(item.url) + '" alt="' + esc(item.name) + '">' : '<video src="' + esc(item.url) + '" controls preload="metadata"></video>';
      return '<figure class="story-media-card">' + preview + '<figcaption>' + esc(item.name) + '</figcaption></figure>';
    }).join('');
    var platforms = metrics.sharePlatforms || {};
    var platformSummary = Object.keys(platforms).sort().map(function (name) { return name + ': ' + Number(platforms[name] || 0); }).join(', ');
    root.innerHTML = '<div class="story-editor-nav"><a class="btn btn--secondary" href="/admin/member-stories/">← All experiences</a><span class="badge">' + esc(story.status) + '</span></div><div class="admin-story-editor-layout"><article class="admin-story-editor-main"><header><h2>' + esc(story.title || '(untitled)') + '</h2><p>Submitted by ' + esc(story.contactName || '') + ' · ' + esc(story.contactEmail || '') + '</p></header><section class="dashboard-panel"><h3>Story media</h3><div class="story-media-previews admin-story-media-previews">' + (media || '<p>No media uploaded.</p>') + '</div></section><label>Published name<input name="displayName" value="' + esc(story.displayName) + '"></label><label>Title<input name="title" value="' + esc(story.title) + '"></label><label>Story (Markdown source)<textarea class="story-markdown-editor" name="bodyMarkdown" rows="20" spellcheck="false">' + esc(story.bodyMarkdown) + '</textarea><span class="form-hint">Edit the Markdown source directly. Uploaded media uses <code>![alt text](story-media:id)</code>.</span></label><label>Message to member<textarea name="memberMessage" rows="4">' + esc(story.memberMessage || '') + '</textarea></label><label>Private admin notes<textarea name="adminNotes" rows="5">' + esc(story.adminNotes || '') + '</textarea></label><section class="dashboard-panel"><h3>Permissions and analysis</h3><p><strong>Permissions:</strong> website ' + (story.consent.website ? 'yes' : 'no') + ', third party ' + (story.consent.thirdParty ? 'yes' : 'no') + ', copyright ' + (story.consent.copyright ? 'yes' : 'no') + ', truthful ' + (story.consent.truthful ? 'yes' : 'no') + '.</p>' + (story.analysis ? '<p><strong>Sentiment:</strong> ' + esc(analysis.sentiment) + '</p><p><strong>Keywords:</strong> ' + esc((analysis.keywords || []).join(', ')) + '</p><p><strong>Analysis prompt:</strong> ' + esc(analysis.promptVersion) + '</p>' : '<p class="form-hint">No editorial analysis yet.</p>') + '</section><section class="dashboard-panel"><h3>Reach</h3><p>' + Number(metrics.readers || 0) + ' readers · ' + Number(metrics.views || 0) + ' views · ' + Number(metrics.engaged || 0) + ' engaged · ' + Number(metrics.ctas || 0) + ' CTA clicks · ' + Number(metrics.shares || 0) + ' share link clicks</p>' + (platformSummary ? '<p><strong>Share clicks by platform:</strong> ' + esc(platformSummary) + '</p>' : '') + '</section><div class="cluster admin-story-editor-actions"><button class="btn btn--secondary" data-action="analyse">Run AI analysis</button><button class="btn btn--primary" data-action="save">Save edit</button><button class="btn btn--secondary" data-action="request-changes">Request clarification</button><button class="btn btn--primary" data-action="approve">Approve</button><button class="btn btn--primary" data-action="publish">Publish</button><button class="btn btn--danger" data-action="archive">Archive</button></div><p data-status role="status" aria-live="polite">' + esc(message || '') + '</p></article></div>';
    root.querySelectorAll('[data-action]').forEach(function (button) {
      button.addEventListener('click', function () {
        var status = root.querySelector('[data-status]');
        root.querySelectorAll('[data-action]').forEach(function (item) { item.disabled = true; });
        status.textContent = 'Working…';
        var payload = {
          id: story.id,
          action: button.dataset.action,
          title: root.querySelector('[name=title]').value,
          displayName: root.querySelector('[name=displayName]').value,
          bodyMarkdown: root.querySelector('[name=bodyMarkdown]').value,
          memberMessage: root.querySelector('[name=memberMessage]').value,
          adminNotes: root.querySelector('[name=adminNotes]').value
        };
        api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) }).then(function () { load('Changes saved.'); }).catch(function (error) {
          status.textContent = error.message;
          root.querySelectorAll('[data-action]').forEach(function (item) { item.disabled = false; });
        });
      });
    });
  }
  document.addEventListener('admin:data', function () { load(); });
  document.addEventListener('identity:login', function () { load(); });
  setTimeout(function () { load(); }, 0);
})();
