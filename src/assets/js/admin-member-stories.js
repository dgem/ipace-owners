(function () {
  'use strict';
  var root = document.querySelector('[data-admin-stories]');
  if (!root) return;
  var state = { page: 1, pageSize: 20, selected: new Set(), data: {}, stories: [] };
  function esc(value) { var node = document.createElement('div'); node.textContent = value == null ? '' : String(value); return node.innerHTML; }
  function api(options) {
    return window.ipaceGetIdentityToken().then(function (token) {
      options = options || {};
      options.headers = Object.assign({ Authorization: 'Bearer ' + token }, options.headers || {});
      return fetch('/api/admin/stories' + (options.query || ''), options);
    }).then(function (response) { return response.json().then(function (data) { if (!response.ok) throw new Error(data.error || 'Request failed'); return data; }); });
  }
  function row(story) {
    var media = (story.media || []).find(function (item) { return item.url; });
    var preview = media ? (media.contentType.indexOf('image/') === 0 ? '<img src="' + esc(media.url) + '" alt="" loading="lazy">' : '<video src="' + esc(media.url) + '" muted preload="metadata"></video>') : '<span aria-hidden="true">▤</span>';
    var editorURL = '/admin/member-stories/edit/?id=' + encodeURIComponent(story.id);
    return '<tr><td><input type="checkbox" data-select="' + esc(story.id) + '" ' + (state.selected.has(story.id) ? 'checked' : '') + ' aria-label="Select ' + esc(story.title) + '"></td><td><div class="admin-story-row"><a class="admin-story-row__preview" href="' + editorURL + '" aria-label="Edit ' + esc(story.title || 'untitled story') + '">' + preview + '</a><div><a class="admin-story-row__title" href="' + editorURL + '">' + esc(story.title || '(untitled)') + '</a><p>' + esc((story.bodyMarkdown || '').slice(0, 220)) + '</p><a class="btn btn--secondary btn--sm" href="' + editorURL + '">Open editorial page</a></div></div></td><td>' + esc(story.displayName || story.contactName || '') + '</td><td><span class="badge">' + esc(story.status) + '</span></td><td>' + esc(story.updatedAt || '') + '</td></tr>';
  }
  function list() {
    var query = new URLSearchParams({ page: state.page, pageSize: state.pageSize, q: (root.querySelector('[name=q]') || {}).value || '', status: (root.querySelector('[name=status]') || {}).value || '', sort: (root.querySelector('[name=sort]') || {}).value || 'updatedAt', direction: (root.querySelector('[name=direction]') || {}).value || 'desc' });
    api({ query: '?' + query }).then(function (data) { state.data = data; state.stories = data.stories || []; render(); }).catch(function (error) { root.innerHTML = '<div class="callout callout--warning"><p>' + esc(error.message) + '</p></div>'; });
  }
  function render() {
    var data = state.data || {}, total = data.total || 0;
    root.innerHTML = '<div class="admin-story-toolbar"><div class="cluster"><label>Search <input name="q" type="search" placeholder="Title, name or email"></label><label>Status <select name="status"><option value="">All statuses</option><option>submitted</option><option>editing</option><option>needs-member-clarification</option><option>approved</option><option>published</option><option>archived</option></select></label><label>Sort <select name="sort"><option value="updatedAt">Recently updated</option><option value="title">Title</option><option value="displayName">Published name</option><option value="status">Status</option></select></label><select name="direction" aria-label="Sort direction"><option value="desc">Descending</option><option value="asc">Ascending</option></select><button class="btn btn--secondary" data-refresh>Apply filters</button></div><div class="cluster admin-story-bulk-actions"><span data-selected-count>0 selected</span><button class="btn btn--secondary btn--sm" data-bulk-action="approve" disabled>Approve selected</button><button class="btn btn--primary btn--sm" data-bulk-action="publish" disabled>Publish selected</button><button class="btn btn--danger btn--sm" data-bulk-action="archive" disabled>Archive selected</button></div></div><div class="data-table-container"><table class="data-table"><thead><tr><th><input type="checkbox" data-select-all aria-label="Select visible stories"></th><th>Story preview</th><th>Member</th><th>Status</th><th>Updated</th></tr></thead><tbody>' + (state.stories.map(row).join('') || '<tr><td colspan="5">No experiences match these filters.</td></tr>') + '</tbody></table></div><div class="cluster admin-story-pagination"><button class="btn btn--secondary" data-page="previous" ' + (data.hasPrevious ? '' : 'disabled') + '>Previous</button><span>Page ' + (data.page || 1) + ' of ' + Math.max(1, Math.ceil(total / (data.pageSize || 20))) + ' · ' + total + ' stories</span><button class="btn btn--secondary" data-page="next" ' + (data.hasNext ? '' : 'disabled') + '>Next</button></div>';
    bind();
    updateSelection();
  }
  function bind() {
    root.querySelector('[data-refresh]').onclick = function () { state.page = 1; list(); };
    root.querySelector('[data-select-all]').onchange = function (event) { root.querySelectorAll('[data-select]').forEach(function (item) { item.checked = event.target.checked; event.target.checked ? state.selected.add(item.dataset.select) : state.selected.delete(item.dataset.select); }); updateSelection(); };
    root.querySelectorAll('[data-select]').forEach(function (item) { item.onchange = function () { item.checked ? state.selected.add(item.dataset.select) : state.selected.delete(item.dataset.select); updateSelection(); }; });
    root.querySelectorAll('[data-page]').forEach(function (item) { item.onclick = function () { state.page += item.dataset.page === 'next' ? 1 : -1; list(); }; });
    root.querySelectorAll('[data-bulk-action]').forEach(function (button) { button.onclick = function () { var action = button.dataset.bulkAction; button.disabled = true; api({ method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ action: action, ids: Array.from(state.selected) }) }).then(function (result) { if (result.failed && result.failed.length) alert(action + ' could not be applied to ' + result.failed.length + ' selected stories. Check their status, consent and required fields.'); list(); }).catch(function (error) { alert(error.message); updateSelection(); }); }; });
  }
  function updateSelection() { var count = root.querySelector('[data-selected-count]'); if (count) count.textContent = state.selected.size + ' selected'; root.querySelectorAll('[data-bulk-action]').forEach(function (button) { button.disabled = !state.selected.size; }); }
  document.addEventListener('admin:data', list);
  document.addEventListener('identity:login', list);
  setTimeout(list, 0);
})();
