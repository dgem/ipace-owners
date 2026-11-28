(function () {
  'use strict';
  var root = document.querySelector('[data-admin-member-search]');
  if (!root) return;
  var results = root.querySelector('[data-results]');
  var status = root.querySelector('[data-status]');
  var previous = root.querySelector('[data-previous]');
  var next = root.querySelector('[data-next]');
  var offset = 0;
  function esc(value) { var node = document.createElement('div'); node.textContent = value == null ? '' : String(value); return node.innerHTML; }
  function request(url, options) {
    return window.ipaceGetIdentityToken().then(function (token) {
      options = options || {}; options.headers = Object.assign({}, options.headers || {}, { Authorization: 'Bearer ' + token });
      return fetch(url, options).then(function (response) { return response.json().then(function (body) { if (!response.ok) throw new Error(body.error || 'Request failed'); return body; }); });
    });
  }
  function search() {
    status.textContent = 'Loading members…';
    var params = new URLSearchParams({ q: root.querySelector('#admin-member-query').value.trim(), preference: root.querySelector('#admin-member-preference').value, country: root.querySelector('#admin-member-country').value, offset: String(offset) });
    request('/api/admin/members?' + params.toString()).then(function (data) {
      results.innerHTML = (data.results || []).map(function (m) {
        return '<article class="card"><h2>' + esc(m.name || 'Unnamed member') + '</h2><p>' + esc(m.email) + '</p><p>Country: ' + esc(m.country || 'Not set') + '</p><p>Cars: ' + esc((m.registrations || []).join(', ') || 'None recorded') + '</p><p>Expertise: ' + esc(m.expertise || 'None recorded') + '</p><p>Interests: ' + esc([m.willingToHelp ? 'Volunteering' : '', m.pressStory ? 'Press' : '', m.betaTester ? 'Feature testing' : ''].filter(Boolean).join(', ') || 'None recorded') + '</p>' + (m.betaTester ? (m.previewApproved == null ? '<p>Preview access status unavailable; check the member has signed in.</p>' : '<p>Preview access: ' + (m.previewApproved ? 'Approved' : 'Not approved') + '</p><button type="button" class="btn btn--secondary" data-approve-tester data-approved="' + (!m.previewApproved) + '" data-uid="' + esc(m.identityUserId) + '" data-email="' + esc(m.email) + '">' + (m.previewApproved ? 'Revoke feature preview' : 'Approve feature preview') + '</button>') : '') + '</article>';
      }).join('') || '<p>No matching members.</p>';
      status.textContent = data.total + (data.total === 1 ? ' matching member' : ' matching members'); previous.hidden = offset === 0; next.hidden = !data.hasMore;
    }).catch(function (error) { status.textContent = error.message; });
  }
  root.querySelector('form').addEventListener('submit', function (event) { event.preventDefault(); offset = 0; search(); });
  previous.addEventListener('click', function () { offset = Math.max(0, offset - 50); search(); });
  next.addEventListener('click', function () { offset += 50; search(); });
  results.addEventListener('click', function (event) {
    var button = event.target.closest('[data-approve-tester]'); if (!button) return;
    button.disabled = true;
    request('/api/admin/feature-testers', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ uid: button.dataset.uid, email: button.dataset.email, approved: button.dataset.approved === 'true' }) }).then(function (data) { search(); status.textContent = data.message; }).catch(function (error) { button.disabled = false; status.textContent = error.message; });
  });
}());
