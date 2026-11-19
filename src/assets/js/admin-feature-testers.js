(function () {
  'use strict';
  var root = document.querySelector('[data-admin-feature-testers]');
  if (!root) return;
  var status = root.querySelector('[data-feature-testers-status]');
  var results = root.querySelector('[data-feature-testers-results]');
  function esc(value) { var node = document.createElement('div'); node.textContent = value == null ? '' : String(value); return node.innerHTML; }
  function request(url, options) { return window.ipaceGetIdentityToken().then(function (token) { options = options || {}; options.headers = Object.assign({}, options.headers || {}, { Authorization: 'Bearer ' + token }); return fetch(url, options).then(function (response) { return response.json().then(function (body) { if (!response.ok) throw new Error(body.error || 'Request failed'); return body; }); }); }); }
  function render(members) { results.innerHTML = members.map(function (member) { return '<article class="card"><h3>' + esc(member.name || 'Unnamed member') + '</h3><p>' + esc(member.email) + '</p><p>' + esc(member.country || '') + '</p><button class="btn btn--primary btn--sm" data-approve-feature-tester data-uid="' + esc(member.identityUserId) + '">Approve member-experiences preview</button><p class="form-hint">They opted in to feature testing. Approval is separate from their preference.</p></article>'; }).join('') || '<p>No opted-in members found.</p>'; }
  root.querySelector('[data-load-feature-testers]').addEventListener('click', function () { status.textContent = 'Loading…'; request('/api/admin/members?q=feature').then(function (data) { render((data.results || []).filter(function (member) { return member.betaTester && member.identityUserId; })); status.textContent = ''; }).catch(function (error) { status.textContent = error.message; }); });
  results.addEventListener('click', function (event) { var button = event.target.closest('[data-approve-feature-tester]'); if (!button) return; button.disabled = true; request('/api/admin/feature-testers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ uid: button.dataset.uid, approved: true }) }).then(function (data) { button.textContent = 'Approved'; status.textContent = data.message; }).catch(function (error) { button.disabled = false; status.textContent = error.message; }); });
}());
