(function () {
  'use strict';
  var root = document.querySelector('[data-admin-stories]');
  if (!root) return;
  var state = { page: 1, pageSize: 20, selected: new Set(), current: null, data: {}, stories: [] };
  function esc(v) { var d = document.createElement('div'); d.textContent = v == null ? '' : String(v); return d.innerHTML; }
  function api(options) {
    return window.ipaceGetIdentityToken().then(function (token) {
      options = options || {};
      options.headers = Object.assign({ Authorization: 'Bearer ' + token }, options.headers || {});
      return fetch('/api/admin/stories' + (options.query || ''), options);
    }).then(function (response) { return response.json().then(function (data) { if (!response.ok) throw new Error(data.error || 'Request failed'); return data; }); });
  }
  function selectedStory() { return state.stories.find(function (story) { return story.id === state.current; }); }
  function showDetail(story) {
    if (!story) return;
    var panel = root.querySelector('[data-story-detail]');
    root.querySelector('[data-story-list-view]').hidden = true;
    panel.hidden = false;
    panel.innerHTML = detail(story);
    bindDetail(story);
    panel.querySelector('[data-back-to-stories]').focus();
  }
  function list() {
    var p = new URLSearchParams({ page: state.page, pageSize: state.pageSize, q: (root.querySelector('[name=q]') || {}).value || '', status: (root.querySelector('[name=status]') || {}).value || '', sort: (root.querySelector('[name=sort]') || {}).value || 'updatedAt', direction: (root.querySelector('[name=direction]') || {}).value || 'desc' });
    state.filters = Object.fromEntries(p);
    api({ query: '?' + p }).then(function (data) { state.data = data; state.stories = data.stories || []; render(); showDetail(selectedStory()); }).catch(function (error) { root.textContent = error.message; });
  }
  function displayDate(value) { var date = new Date(value); return value && !isNaN(date.getTime()) ? date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' }) : ''; }
  function row(story) { return '<tr><td><input type="checkbox" data-select="' + esc(story.id) + '" ' + (state.selected.has(story.id) ? 'checked' : '') + ' aria-label="Select ' + esc(story.title) + '"></td><td><button class="link-button" data-open="' + esc(story.id) + '">' + esc(story.title || '(untitled)') + '</button></td><td>' + esc(story.displayName || story.contactName || '') + '</td><td><span class="badge">' + esc(story.status) + '</span></td><td>' + esc(displayDate(story.updatedAt)) + '</td></tr>'; }
  function mediaMarkdown(item) {
    var isVideo = (item.contentType || '').indexOf('video/') === 0;
    var alt = isVideo ? 'Video' : (item.name || 'Image');
    return '![' + alt.replace(/[\[\]]/g, '') + '](story-media:' + item.id + ')';
  }
  function mediaAssets(story) {
    var items = (story.media || []).filter(function (item) { return item.url && item.id; });
    if (!items.length) return '';
    return '<section class="admin-story-media-assets" aria-label="Attached media"><h3>Attached media</h3><p class="form-hint">Copy a Markdown reference and paste it into the story source. The reference stays valid; temporary upload URLs do not.</p><div class="admin-story-media-assets__grid">' + items.map(function (item) {
      var preview = (item.contentType || '').indexOf('image/') === 0 ? '<img src="' + esc(item.url) + '" alt="' + esc(item.name) + '" class="story-review-media">' : '<video src="' + esc(item.url) + '" controls class="story-review-media"></video>';
      var markdown = mediaMarkdown(item);
      return '<article class="admin-story-media-asset">' + preview + '<p>' + esc(item.name || 'Uploaded media') + '</p><label>Markdown reference<input readonly value="' + esc(markdown) + '" aria-label="Markdown reference for ' + esc(item.name || 'uploaded media') + '"></label><button class="btn btn--secondary btn--sm" type="button" data-copy-media-markdown>Copy Markdown</button></article>';
    }).join('') + '</div><p data-media-copy-status role="status"></p></section>';
  }
  function render() {
    var data = state.data || {}, total = data.total || 0;
    root.innerHTML = '<div data-story-list-view><div class="admin-story-toolbar"><div class="cluster"><label>Search <input name="q" type="search" placeholder="Title, name or email"></label><label>Status <select name="status"><option value="">All statuses</option><option>draft</option><option>submitted</option><option>editing</option><option>needs-member-clarification</option><option>approved</option><option>published</option><option>archived</option><option>deleted</option></select></label><label>Sort <select name="sort"><option value="updatedAt">Recently updated</option><option value="title">Title</option><option value="displayName">Published name</option><option value="status">Status</option></select></label><select name="direction" aria-label="Sort direction"><option value="desc">Descending</option><option value="asc">Ascending</option></select><button class="btn btn--secondary" data-refresh>Apply filters</button></div><div class="cluster admin-story-bulk-actions"><span data-selected-count>0 selected</span><button class="btn btn--secondary btn--sm" data-bulk-action="approve" disabled>Approve selected</button><button class="btn btn--primary btn--sm" data-bulk-action="publish" disabled>Publish selected</button><button class="btn btn--secondary btn--sm" data-bulk-action="archive" disabled>Archive selected</button><button class="btn btn--danger btn--sm" data-show-bulk-delete disabled>Delete selected</button></div><form class="admin-story-bulk-delete" data-confirm-bulk-delete hidden><p>Delete the selected experiences? They will be removed from the website and retained for recovery.</p><label>Type DELETE to delete selected experiences <input name="bulkDeleteConfirmation" autocomplete="off" required></label><button class="btn btn--danger btn--sm" type="submit">Confirm selected deletion</button><button class="btn btn--secondary btn--sm" type="button" data-cancel-bulk-delete>Cancel</button><p data-bulk-delete-status role="status"></p></form></div><div class="admin-story-workspace"><section><div class="data-table-container"><table class="data-table"><thead><tr><th><input type="checkbox" data-select-all aria-label="Select visible stories"></th><th>Story</th><th>Member</th><th>Status</th><th>Updated</th></tr></thead><tbody>' + (state.stories.map(row).join('') || '<tr><td colspan="5">No experiences match these filters.</td></tr>') + '</tbody></table></div><div class="cluster admin-story-pagination"><button class="btn btn--secondary" data-page="previous" ' + (data.hasPrevious ? '' : 'disabled') + '>Previous</button><span>Page ' + (data.page || 1) + ' of ' + Math.max(1, Math.ceil(total / (data.pageSize || 20))) + ' · ' + total + ' stories</span><button class="btn btn--secondary" data-page="next" ' + (data.hasNext ? '' : 'disabled') + '>Next</button></div></section></div></div><section class="dashboard-panel admin-story-detail" data-story-detail hidden></section>';
    ['q', 'status', 'sort', 'direction'].forEach(function (name) { if (state.filters) root.querySelector('[name="' + name + '"]').value = state.filters[name]; });
    bind(); updateSelection();
  }
  function detail(story) {
    var analysis = story.analysis || {}, impact = (state.data.impact || {})[story.id] || {};
    var media = mediaAssets(story);
    var platforms = impact.sharePlatforms || {};
    var platformSummary = Object.keys(platforms).sort().map(function (name) { return name + ': ' + Number(platforms[name] || 0); }).join(', ');
    return '<button class="btn btn--secondary btn--sm admin-story-back" type="button" data-back-to-stories>← Back to experiences</button><div class="dashboard-panel__header"><div><h2>' + esc(story.title) + '</h2><p>' + esc(story.contactName || '') + ' · ' + esc(story.contactEmail || '') + '</p></div><span class="badge">' + esc(story.status) + '</span></div><p><strong>Impact:</strong> ' + Number(impact.readers || 0) + ' readers, ' + Number(impact.views || 0) + ' views, ' + Number(impact.engaged || 0) + ' engaged, ' + Number(impact.ctas || 0) + ' CTA clicks, ' + Number(impact.shares || 0) + ' share link clicks.</p>' + (platformSummary ? '<p><strong>Share clicks by platform:</strong> ' + esc(platformSummary) + '</p>' : '') + media + '<label>Published name<input name="displayName" value="' + esc(story.displayName) + '"></label><label>Title<input name="title" value="' + esc(story.title) + '"></label><label>Story (Markdown source)<textarea class="story-markdown-editor" name="bodyMarkdown" rows="14" spellcheck="false">' + esc(story.bodyMarkdown) + '</textarea><span class="form-hint">Edit the Markdown source directly. Uploaded media uses <code>![alt text](story-media:id)</code>.</span></label><button class="btn btn--secondary" type="button" data-preview-story aria-expanded="false">Preview story</button><section class="story-preview" data-story-preview hidden aria-label="Story preview"></section><label>Message to member<textarea name="memberMessage" rows="3">' + esc(story.memberMessage || '') + '</textarea></label><label>Private admin notes<textarea name="adminNotes" rows="4">' + esc(story.adminNotes || '') + '</textarea></label><p><strong>Permissions:</strong> website ' + (story.consent.website ? 'yes' : 'no') + ', third party ' + (story.consent.thirdParty ? 'yes' : 'no') + ', copyright ' + (story.consent.copyright ? 'yes' : 'no') + ', truthful ' + (story.consent.truthful ? 'yes' : 'no') + '.</p>' + (story.analysis ? '<details open><summary>Editorial analysis · ' + esc(analysis.promptVersion) + '</summary><p><strong>Sentiment:</strong> ' + esc(analysis.sentiment) + '</p><p><strong>Keywords:</strong> ' + esc((analysis.keywords || []).join(', ')) + '</p></details>' : '<p class="form-hint">No editorial analysis yet.</p>') + (story.status === 'deleted' ? '<p class="form-hint">This experience has been deleted.</p>' : '<div class="cluster"><button class="btn btn--primary" data-action="analyse">Run AI analysis</button><button class="btn btn--primary" data-action="save">Save edit</button><button class="btn btn--primary" data-action="request-changes">Request clarification</button><button class="btn btn--primary" data-action="approve">Approve</button><button class="btn btn--primary" data-action="publish">Publish</button><button class="btn btn--primary" data-action="archive">Archive</button><button class="btn btn--danger" type="button" data-delete-story>Delete</button></div><form data-confirm-delete hidden><p>Delete this experience? It will be removed from the website and retained for recovery.</p><label>Type DELETE to confirm <input name="deleteConfirmation" autocomplete="off" required></label><button class="btn btn--danger btn--sm" type="submit">Confirm deletion</button><button class="btn btn--secondary btn--sm" type="button" data-cancel-delete>Cancel</button></form><p data-status role="status"></p>');
  }
  function bind() {
    root.querySelector('[data-refresh]').onclick = function () { state.page = 1; list(); };
    root.querySelectorAll('[data-open]').forEach(function (item) { item.onclick = function () { state.current = item.dataset.open; showDetail(selectedStory()); }; });
    root.querySelectorAll('[data-select]').forEach(function (item) { item.onchange = function () { item.checked ? state.selected.add(item.dataset.select) : state.selected.delete(item.dataset.select); updateSelection(); }; });
    root.querySelector('[data-select-all]').onchange = function (event) { root.querySelectorAll('[data-select]').forEach(function (item) { item.checked = event.target.checked; event.target.checked ? state.selected.add(item.dataset.select) : state.selected.delete(item.dataset.select); }); updateSelection(); };
    root.querySelectorAll('[data-page]').forEach(function (item) { item.onclick = function () { state.page += item.dataset.page === 'next' ? 1 : -1; list(); }; });
    var bulkDeleteButton = root.querySelector('[data-show-bulk-delete]');
    var bulkDeleteForm = root.querySelector('[data-confirm-bulk-delete]');
    bulkDeleteButton.onclick = function () { bulkDeleteForm.hidden = false; bulkDeleteForm.querySelector('input').focus(); };
    bulkDeleteForm.querySelector('[data-cancel-bulk-delete]').onclick = function () { bulkDeleteForm.hidden = true; bulkDeleteButton.focus(); };
    bulkDeleteForm.onsubmit = function (event) {
      event.preventDefault();
      var confirmation = bulkDeleteForm.querySelector('input').value;
      var status = bulkDeleteForm.querySelector('[data-bulk-delete-status]');
      if (confirmation !== 'DELETE') { status.textContent = 'Type DELETE to confirm.'; return; }
      var submit = bulkDeleteForm.querySelector('[type="submit"]');
      submit.disabled = true;
      api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: 'delete', ids: Array.from(state.selected), confirmation: confirmation }) }).then(function (result) { if (result.failed && result.failed.length) alert('delete could not be applied to ' + result.failed.length + ' selected stories.'); state.selected.clear(); list(); }).catch(function (error) { submit.disabled = false; status.textContent = error.message; });
    };
    root.querySelectorAll('[data-bulk-action]').forEach(function (button) { button.onclick = function () { var action = button.dataset.bulkAction; button.disabled = true; api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: action, ids: Array.from(state.selected) }) }).then(function (result) { if (result.failed && result.failed.length) alert(action + ' could not be applied to ' + result.failed.length + ' selected stories. Check their status, consent and required fields.'); list(); }).catch(function (error) { alert(error.message); updateSelection(); }); }; });
  }
  function bindDetail(story) {
    var panel = root.querySelector('[data-story-detail]');
    function editorValues() {
      return ['title', 'displayName', 'bodyMarkdown', 'memberMessage', 'adminNotes'].map(function (name) { return panel.querySelector('[name="' + name + '"]').value; });
    }
    var initialValues = JSON.stringify(editorValues());
    panel.querySelector('[data-back-to-stories]').onclick = function () {
      if (JSON.stringify(editorValues()) !== initialValues && !window.confirm('Discard your unsaved changes and return to experiences?')) return;
      state.current = null;
      panel.hidden = true;
      root.querySelector('[data-story-list-view]').hidden = false;
      var rowButton = root.querySelector('[data-open="' + story.id + '"]');
      if (rowButton) rowButton.focus();
    };
    var previewButton = panel.querySelector('[data-preview-story]');
    var preview = panel.querySelector('[data-story-preview]');
    previewButton.onclick = function () {
      if (!preview.hidden) { preview.hidden = true; previewButton.setAttribute('aria-expanded', 'false'); previewButton.textContent = 'Preview story'; preview.querySelectorAll('video').forEach(function (video) { video.pause(); }); return; }
      preview.innerHTML = '<p class="form-hint">Preview of your current edits · changes are not saved</p><article class="member-story"><p class="eyebrow">Owners’ story</p><h2>' + esc(panel.querySelector('[name=title]').value) + '</h2><p class="lede">By ' + esc(panel.querySelector('[name=displayName]').value) + '</p><div class="story-markdown">' + window.ipaceStoryRenderer.markdown(panel.querySelector('[name=bodyMarkdown]').value, story.media) + '</div><section class="story-media-gallery" aria-label="Attached media">' + window.ipaceStoryRenderer.mediaGallery(story.media) + '</section></article>';
      preview.hidden = false; previewButton.setAttribute('aria-expanded', 'true'); previewButton.textContent = 'Close preview';
    };
    panel.querySelectorAll('[data-copy-media-markdown]').forEach(function (button) { button.onclick = function () {
      var input = button.previousElementSibling.querySelector('input');
      var status = panel.querySelector('[data-media-copy-status]');
      function copied() { status.textContent = 'Markdown reference copied.'; button.textContent = 'Copied'; }
      input.select();
      if (navigator.clipboard && navigator.clipboard.writeText) navigator.clipboard.writeText(input.value).then(copied).catch(function () { document.execCommand('copy'); copied(); });
      else { document.execCommand('copy'); copied(); }
    }; });
    var deleteButton = panel.querySelector('[data-delete-story]');
    if (deleteButton) {
      var form = panel.querySelector('[data-confirm-delete]');
      deleteButton.onclick = function () { form.hidden = false; form.querySelector('input').focus(); };
      form.querySelector('[data-cancel-delete]').onclick = function () { form.hidden = true; deleteButton.focus(); };
      form.onsubmit = function (event) {
        event.preventDefault();
        var status = panel.querySelector('[data-status]');
        var confirmation = form.querySelector('input').value;
        if (confirmation !== 'DELETE') { status.textContent = 'Type DELETE to confirm.'; return; }
        var submit = form.querySelector('[type="submit"]');
        submit.disabled = true;
        api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: story.id, action: 'delete', confirmation: confirmation }) }).then(function () { state.selected.delete(story.id); state.current = null; list(); }).catch(function (error) { submit.disabled = false; status.textContent = error.message; });
      };
    }
    panel.querySelectorAll('[data-action]').forEach(function (button) { button.onclick = function () { var status = panel.querySelector('[data-status]'); status.textContent = 'Working…'; api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: story.id, action: button.dataset.action, title: panel.querySelector('[name=title]').value, displayName: panel.querySelector('[name=displayName]').value, bodyMarkdown: panel.querySelector('[name=bodyMarkdown]').value, memberMessage: panel.querySelector('[name=memberMessage]').value, adminNotes: panel.querySelector('[name=adminNotes]').value }) }).then(function () { status.textContent = 'Saved.'; list(); }).catch(function (error) { status.textContent = error.message; }); }; });
  }
  function updateSelection() { var count = root.querySelector('[data-selected-count]'); if (count) count.textContent = state.selected.size + ' selected'; root.querySelectorAll('[data-bulk-action], [data-show-bulk-delete]').forEach(function (button) { button.disabled = !state.selected.size; }); }
  document.addEventListener('admin:data', list); document.addEventListener('identity:login', list); setTimeout(list, 0);
})();
