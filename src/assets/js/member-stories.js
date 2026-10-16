(function () {
  'use strict';

  var root = document.querySelector('[data-member-stories]');
  if (!root) return;
  var state = null;

  function esc(value) { var node = document.createElement('div'); node.textContent = value == null ? '' : String(value); return node.innerHTML; }
  function checked(value) { return value ? ' checked' : ''; }
  function token() { return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : Promise.reject(new Error('Sign in required')); }
  function api(path, options) {
    return token().then(function (idToken) {
      options = options || {};
      options.headers = Object.assign({ 'Authorization': 'Bearer ' + idToken }, options.headers || {});
      return fetch(path, options);
    }).then(function (response) { return response.json().then(function (data) { if (!response.ok) throw new Error(data.error || 'Request failed'); return data; }); });
  }
  function load() {
    return api('/api/member/stories').then(function (data) { state = data; render(); }).catch(function (error) { root.innerHTML = '<div class="callout callout--warning"><p>' + esc(error.message) + '</p></div>'; });
  }
  function option(value, label, selected) { return '<option value="' + esc(value) + '"' + (selected ? ' selected' : '') + '>' + esc(label) + '</option>'; }

  function render(editing) {
    if (!state.eligible) {
      root.innerHTML = '<div class="callout callout--info"><h2>Complete your member record first</h2><p>A story needs your name and at least one registered car so it can contribute to the evidence base.</p><p><a class="btn btn--primary" href="/member/account/">Add contact details</a> <a class="btn btn--secondary" href="/member/submit-vehicle-data/">Register a car</a></p></div>';
      return;
    }
    editing = editing || {};
    var evidence = editing.evidence || {};
    var consent = editing.consent || {};
    var vehicles = (state.vehicles || []).map(function (item) {
      var vehicle = item.vehicle || {};
      return option(item.id, (vehicle.registration || 'Vehicle') + (vehicle.modelYear ? ' · ' + vehicle.modelYear : ''), evidence.vehicleId === item.id);
    }).join('');
    var submissions = (state.stories || []).map(function (story) {
      var editable = story.status === 'draft' || story.status === 'needs-member-clarification';
      var message = story.memberMessage ? '<p class="callout callout--info"><strong>Admin request:</strong> ' + esc(story.memberMessage) + '</p>' : '';
      return '<li><div><strong>' + esc(story.title) + '</strong> <span class="badge">' + esc(story.status) + '</span>' + message + '</div><div class="cluster">' +
        (editable ? '<button class="btn btn--secondary btn--sm" type="button" data-edit-story="' + esc(story.id) + '">Edit story</button>' : '') +
        (story.status !== 'withdrawal-requested' ? '<button class="btn btn--secondary btn--sm" type="button" data-withdraw-story="' + esc(story.id) + '">Withdraw permissions</button>' : '') + '</div></li>';
    }).join('');
    var existingMedia = (editing.media || []).filter(function (item) { return item.status === 'ready'; }).map(function (item) { return '<li>' + esc(item.name) + '</li>'; }).join('');

    root.innerHTML = '<section class="story-guidance"><h2>Your experience can strengthen the evidence</h2><p>Link the records behind your story. If anything is missing, add it to your <a href="/member/dashboard/">vehicle history</a> first.</p></section>' +
      '<form class="member-form-panel story-form" data-story-form data-story-id="' + esc(editing.id || '') + '"><h2>' + (editing.id ? 'Edit your story' : 'Write your story') + '</h2><div class="member-form-grid">' +
      '<div class="form-group member-form-grid__wide"><label for="story-title">Story title</label><input id="story-title" name="title" required maxlength="180" value="' + esc(editing.title || '') + '"></div>' +
      '<div class="form-group"><label for="story-name">Published name</label><input id="story-name" name="displayName" required maxlength="160" value="' + esc(editing.displayName || state.profileName) + '"><p class="form-hint">Your saved contact details remain private.</p></div>' +
      '<div class="form-group"><label for="story-vehicle">Vehicle</label><select id="story-vehicle" name="vehicleId" required><option value="">Choose your car</option>' + vehicles + '</select></div>' +
      '<div class="form-group member-form-grid__wide"><label for="story-body">Your story</label><textarea id="story-body" name="bodyMarkdown" rows="16" required minlength="80" maxlength="20000" placeholder="What happened?\n\nHow did it affect you?\n\nWhat response did you receive?\n\nWhat outcome would make a lasting difference?">' + esc(editing.bodyMarkdown || '') + '</textarea><p class="form-hint">Markdown headings, lists, links, bold and italic text are supported. Your wording stays yours.</p></div>' +
      '<fieldset class="member-form-grid__wide"><legend>Records supporting this story</legend><div data-story-evidence><p class="form-hint">Choose a vehicle to see its service, Executive Office and SoH records.</p></div></fieldset>' +
      '<div class="form-group member-form-grid__wide"><label for="story-media">Images and short videos (optional)</label>' + (existingMedia ? '<p>Already uploaded:</p><ul>' + existingMedia + '</ul>' : '') + '<input id="story-media" name="media" type="file" accept="image/jpeg,image/png,image/webp,video/mp4,video/quicktime,video/webm" multiple><p class="form-hint">Up to 5 images and 2 videos; 250 MB total.</p></div></div>' +
      '<fieldset><legend>Permissions and confirmations</legend><label class="check-label"><input name="websiteConsent" type="checkbox" required' + checked(consent.website) + '> I give permission to publish this story and its selected media on this website.</label><label class="check-label"><input name="thirdPartyConsent" type="checkbox"' + checked(consent.thirdParty) + '> I also give permission to use it in social media, press and publisher materials.</label><label class="check-label"><input name="copyrightConfirmed" type="checkbox"' + checked(consent.copyright) + '> I own the copyright, or have permission to supply, every uploaded item.</label><label class="check-label"><input name="truthfulConfirmed" type="checkbox" required' + checked(consent.truthful) + '> This is a true account to the best of my knowledge.</label></fieldset>' +
      '<div class="cluster"><button class="btn btn--primary" type="submit">Submit for review</button><button class="btn btn--secondary" type="button" data-save-story>Save draft</button>' + (editing.id ? '<button class="btn btn--secondary" type="button" data-cancel-edit>Cancel edit</button>' : '') + '</div><p data-story-status role="status" aria-live="polite"></p></form>' +
      (submissions ? '<section class="dashboard-panel"><h2>Your submissions</h2><ul class="story-submission-list">' + submissions + '</ul></section>' : '');

    var vehicleSelect = root.querySelector('[name="vehicleId"]');
    vehicleSelect.addEventListener('change', function () { renderEvidence(vehicleSelect.value, [], []); });
    if (evidence.vehicleId) renderEvidence(evidence.vehicleId, evidence.serviceEventIds || [], evidence.sohReadingIds || []);
    root.querySelector('[data-story-form]').addEventListener('submit', function (event) { event.preventDefault(); save(true); });
    root.querySelector('[data-save-story]').addEventListener('click', function () { save(false); });
    var cancel = root.querySelector('[data-cancel-edit]');
    if (cancel) cancel.addEventListener('click', function () { render(); });
    root.querySelectorAll('[data-edit-story]').forEach(function (button) {
      button.addEventListener('click', function () {
        var story = (state.stories || []).find(function (item) { return item.id === button.dataset.editStory; });
        if (story) { render(story); root.querySelector('[data-story-form]').scrollIntoView({ behavior: 'smooth', block: 'start' }); }
      });
    });
    root.querySelectorAll('[data-withdraw-story]').forEach(function (button) {
      button.addEventListener('click', function () {
        button.disabled = true;
        api('/api/member/story-withdraw', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: button.dataset.withdrawStory }) }).then(load).catch(function (error) { button.disabled = false; button.insertAdjacentText('afterend', ' ' + error.message); });
      });
    });
  }

  function renderEvidence(vehicleId, selectedService, selectedSOH) {
    var service = (state.serviceEvents || []).filter(function (item) { return item.vehicleId === vehicleId; });
    var soh = (state.sohReadings || []).filter(function (item) { return item.vehicleId === vehicleId; });
    var box = root.querySelector('[data-story-evidence]');
    function checks(items, name, selected, labeler) {
      return items.map(function (item) { return '<label class="check-label"><input type="checkbox" name="' + name + '" value="' + esc(item.id) + '"' + checked(selected.indexOf(item.id) !== -1) + '> ' + esc(labeler(item)) + '</label>'; }).join('');
    }
    var serviceHtml = checks(service, 'serviceEventIds', selectedService || [], function (item) { return (item.occurredAt || 'No date') + ' · ' + (item.title || item.eventType || 'Record'); });
    var sohHtml = checks(soh, 'sohReadingIds', selectedSOH || [], function (item) { var battery = item.battery || {}; return (battery.measuredAt || 'No date') + ' · ' + (battery.stateOfHealth == null ? 'SoH reading' : battery.stateOfHealth + '% SoH'); });
    box.innerHTML = '<h3>Service, fault and JLR contacts</h3>' + (serviceHtml || '<p class="form-hint">No records yet. <a href="/member/dashboard/">Add one</a>.</p>') + '<h3>Battery SoH readings</h3>' + (sohHtml || '<p class="form-hint">No readings yet. <a href="/member/dashboard/">Add one</a>.</p>');
  }

  function payload(form, submit) {
    var data = new FormData(form);
    return { id: form.dataset.storyId, title: data.get('title'), displayName: data.get('displayName'), bodyMarkdown: data.get('bodyMarkdown'), vehicleId: data.get('vehicleId'), serviceEventIds: data.getAll('serviceEventIds'), sohReadingIds: data.getAll('sohReadingIds'), websiteConsent: data.has('websiteConsent'), thirdPartyConsent: data.has('thirdPartyConsent'), copyrightConfirmed: data.has('copyrightConfirmed'), truthfulConfirmed: data.has('truthfulConfirmed'), submit: submit };
  }
  function upload(storyId, file) {
    return api('/api/member/story-upload', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ storyId: storyId, name: file.name, contentType: file.type, size: file.size }) }).then(function (reservation) {
      return fetch(reservation.uploadUrl, { method: 'PUT', headers: { 'Content-Type': reservation.contentType }, body: file }).then(function (response) {
        if (!response.ok) throw new Error('Could not upload ' + file.name);
        return api('/api/member/story-upload-complete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ storyId: storyId, mediaId: reservation.mediaId }) });
      });
    });
  }
  function save(submit) {
    var form = root.querySelector('[data-story-form]');
    if (submit && !form.reportValidity()) return;
    var status = root.querySelector('[data-story-status]');
    var data = payload(form, false);
    var files = Array.from(form.elements.media.files || []);
    status.textContent = 'Saving your story…';
    api('/api/member/stories', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }).then(function (result) {
      return files.reduce(function (chain, file) { return chain.then(function () { status.textContent = 'Uploading ' + file.name + '…'; return upload(result.story.id, file); }); }, Promise.resolve()).then(function () {
        if (!submit) return result;
        data.id = result.story.id; data.submit = true;
        return api('/api/member/stories', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
      });
    }).then(function () { status.textContent = submit ? 'Submitted for review.' : 'Draft saved.'; return load(); }).catch(function (error) { status.textContent = error.message; });
  }

  document.addEventListener('member:data', load);
  document.addEventListener('identity:login', load);
  if (window.ipaceIdentityUser) load();
})();
