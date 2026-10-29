(function () {
  'use strict';

  var root = document.querySelector('[data-member-stories]');
  if (!root) return;

  var state = null;
  var selectedFiles = [];
  var previewURLs = [];
  var editorMedia = [];
  var existingMediaUsage = { images: 0, videos: 0, bytes: 0 };
  var mediaTypes = ['image/jpeg', 'image/png', 'image/webp', 'video/mp4', 'video/quicktime', 'video/webm'];
  var imageResizeBytes = 8 * 1024 * 1024;
  var imageResizePixels = 4096;

  function esc(value) {
    var node = document.createElement('div');
    node.textContent = value == null ? '' : String(value);
    return node.innerHTML;
  }

  function checked(value) { return value ? ' checked' : ''; }
  function token() { return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : Promise.reject(new Error('Sign in required')); }
  function api(path, options) {
    return token().then(function (idToken) {
      options = options || {};
      options.headers = Object.assign({ 'Authorization': 'Bearer ' + idToken }, options.headers || {});
      return fetch(path, options);
    }).then(function (response) {
      return response.json().then(function (data) {
        if (!response.ok) {
          var error = new Error(data.error || 'Request failed');
          error.traceId = data.traceId || '';
          throw error;
        }
        return data;
      });
    });
  }

  function displayError(error) {
    var message = error && error.message ? error.message : 'Request failed';
    return error && error.traceId ? message + ' (reference ' + error.traceId + ')' : message;
  }

  function load() {
    var enabled = window.ipaceMemberFeatureEnabled ? window.ipaceMemberFeatureEnabled('member-experiences') : Promise.resolve(true);
    return enabled.then(function (allowed) {
      if (!allowed) return null;
      return api('/api/member/stories');
    }).then(function (data) {
      if (!data) return;
      state = data;
      render();
    }).catch(function (error) {
      root.innerHTML = '<div class="callout callout--warning"><p>' + esc(error.message) + '</p></div>';
    });
  }

  function clearPreviews() {
    previewURLs.forEach(function (url) { URL.revokeObjectURL(url); });
    previewURLs = [];
    selectedFiles = [];
  }

  function inlineMarkdown(value) {
    return esc(value)
      .replace(/!\[([^\]]*)\]\(story-media:([a-zA-Z0-9_-]+)\)/g, function (match, alt, id) {
        var item = editorMedia.find(function (media) { return media.id === id && media.status === 'ready' && media.url; });
        if (!item) return alt;
        if (item.contentType.indexOf('video/') === 0) return '<video src="' + esc(item.url) + '" controls preload="metadata" data-story-media-id="' + esc(item.id) + '"></video>';
        return '<a class="story-inline-image" href="' + esc(item.url) + '" target="_blank" rel="noopener noreferrer"><img src="' + esc(item.url) + '" alt="' + (alt || esc(item.name)) + '" data-story-media-id="' + esc(item.id) + '"></a>';
      })
      .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
      .replace(/\*(.+?)\*/g, '<em>$1</em>');
  }

  function markdownToEditorHTML(value) {
    if (!String(value || '').trim()) return '';
    return String(value || '').trim().split(/\n{2,}/).map(function (block) {
      var lines = block.split('\n');
      if (lines.every(function (line) { return /^[-*] /.test(line); })) {
        return '<ul>' + lines.map(function (line) { return '<li>' + inlineMarkdown(line.replace(/^[-*] /, '')) + '</li>'; }).join('') + '</ul>';
      }
      if (lines.every(function (line) { return /^\d+[.)] /.test(line); })) {
        return '<ol>' + lines.map(function (line) { return '<li>' + inlineMarkdown(line.replace(/^\d+[.)] /, '')) + '</li>'; }).join('') + '</ol>';
      }
      if (lines.every(function (line) { return /^> ?/.test(line); })) {
        return '<blockquote>' + lines.map(function (line) { return inlineMarkdown(line.replace(/^> ?/, '')); }).join('<br>') + '</blockquote>';
      }
      return '<p>' + inlineMarkdown(block.replace(/^#{1,3} /, '')).replace(/\n/g, '<br>') + '</p>';
    }).join('');
  }

  function editorNodeToMarkdown(node) {
    if (node.nodeType === Node.TEXT_NODE) return node.nodeValue || '';
    if (node.nodeType !== Node.ELEMENT_NODE) return '';
    var tag = node.tagName.toLowerCase();
    var children = Array.from(node.childNodes).map(editorNodeToMarkdown).join('');
    if (tag === 'br') return '\n';
    if (tag === 'strong' || tag === 'b') return '**' + children + '**';
    if (tag === 'em' || tag === 'i') return '*' + children + '*';
    if (tag === 'img') {
      var mediaID = node.getAttribute('data-story-media-id') || '';
      var alt = (node.getAttribute('alt') || '').replace(/\[/g, '').replace(/\]/g, '');
      return /^[a-zA-Z0-9_-]+$/.test(mediaID) ? '![' + alt + '](story-media:' + mediaID + ')' : '';
    }
    if (tag === 'video') {
      var videoID = node.getAttribute('data-story-media-id') || '';
      return /^[a-zA-Z0-9_-]+$/.test(videoID) ? '![Video](story-media:' + videoID + ')' : '';
    }
    if (tag === 'a') {
      return children;
    }
    if (tag === 'ul' || tag === 'ol') {
      return Array.from(node.children).filter(function (item) { return item.tagName.toLowerCase() === 'li'; }).map(function (item, index) {
        return (tag === 'ol' ? (index + 1) + '. ' : '- ') + Array.from(item.childNodes).map(editorNodeToMarkdown).join('').trim();
      }).join('\n') + '\n\n';
    }
    if (tag === 'li') return children;
    if (tag === 'blockquote') return children.split('\n').map(function (line) { return '> ' + line.trim(); }).join('\n') + '\n\n';
    if (tag === 'p' || tag === 'div') return children.trim() + '\n\n';
    return children;
  }

  function editorMarkdown() {
    var editor = root.querySelector('[data-story-editor]');
    if (!editor) return '';
    return Array.from(editor.childNodes).map(editorNodeToMarkdown).join('').replace(/\n{3,}/g, '\n\n').trim();
  }

  function statusLabel(value) {
    return ({ draft: 'Draft', submitted: 'Submitted', 'needs-member-clarification': 'Needs more detail', editing: 'Being edited', approved: 'Approved', published: 'Published', 'withdrawal-requested': 'Permission withdrawn' })[value] || value;
  }

  function vehicleOption(item, selected) {
    var vehicle = item.vehicle || {};
    var label = vehicle.registration || (vehicle.vinLast6 ? 'VIN …' + vehicle.vinLast6 : 'Vehicle');
    if (vehicle.modelYear) label += ' · ' + vehicle.modelYear;
    return '<option value="' + esc(item.id) + '"' + (selected ? ' selected' : '') + '>' + esc(label) + '</option>';
  }

  function render(editing) {
    clearPreviews();
    if (!state.eligible) {
      root.innerHTML = '<div class="callout callout--info"><h2>Complete your member record first</h2><p>Add your name and display name, then register at least one I-PACE before sharing an experience.</p><div class="cluster"><a class="btn btn--primary" href="/member/account/">Go to My Data</a><a class="btn btn--secondary" href="/member/submit-vehicle-data/">Register a car</a></div></div>';
      return;
    }

    editing = editing || {};
    editorMedia = editing.media || [];
    existingMediaUsage = editorMedia.filter(function (item) { return item.status === 'ready'; }).reduce(function (usage, item) {
      usage.bytes += Number(item.size) || 0;
      if (item.contentType && item.contentType.indexOf('image/') === 0) usage.images += 1;
      else if (item.contentType && item.contentType.indexOf('video/') === 0) usage.videos += 1;
      return usage;
    }, { images: 0, videos: 0, bytes: 0 });
    var evidence = editing.evidence || {};
    var consent = editing.consent || {};
    var vehicles = state.vehicles || [];
    var selectedVehicle = evidence.vehicleId || (vehicles.length === 1 ? vehicles[0].id : '');
    var vehicleControl = vehicles.length > 1
      ? '<div class="form-group"><label for="story-vehicle">Which car is this about?</label><select id="story-vehicle" name="vehicleId" required><option value="">Choose your car</option>' + vehicles.map(function (item) { return vehicleOption(item, item.id === selectedVehicle); }).join('') + '</select></div>'
      : '<input name="vehicleId" type="hidden" value="' + esc(selectedVehicle) + '">';
    var existingMedia = editorMedia.filter(function (item) { return item.status === 'ready'; }).map(function (item) {
      var legacyImageHook = item.contentType.indexOf('image/') === 0 ? ' data-insert-story-image="' + esc(item.id) + '"' : '';
      var insert = '<button class="btn btn--secondary btn--sm" type="button" data-insert-story-media="' + esc(item.id) + '"' + legacyImageHook + '>Insert ' + (item.contentType.indexOf('image/') === 0 ? 'image' : 'video') + '</button>';
      return '<li><span>' + esc(item.name) + '</span>' + insert + '</li>';
    }).join('');
    var submissions = (state.stories || []).map(function (story) {
      var editable = story.status === 'draft' || story.status === 'submitted' || story.status === 'editing' || story.status === 'needs-member-clarification';
      var message = story.memberMessage ? '<p class="story-member-message"><strong>More detail requested:</strong> ' + esc(story.memberMessage) + '</p>' : '';
      var sentiment = story.analysis && story.analysis.sentiment ? '<p class="form-hint"><strong>Editorial sentiment:</strong> ' + esc(story.analysis.sentiment) + '</p>' : '';
      return '<li><div class="story-submission-list__summary"><strong>' + esc(story.title) + '</strong><span class="badge">' + esc(statusLabel(story.status)) + '</span></div>' + message + '<div class="cluster">' +
        sentiment +
        (editable ? '<button class="btn btn--secondary btn--sm" type="button" data-edit-story="' + esc(story.id) + '">Edit</button>' : '') +
        (story.status !== 'draft' && story.status !== 'withdrawal-requested' ? '<button class="btn btn--secondary btn--sm" type="button" data-withdraw-story="' + esc(story.id) + '">Withdraw permission</button>' : '') + '</div></li>';
    }).join('');

    root.innerHTML = '<section class="story-guidance"><div><h2>Share your experience</h2><p>Tell us what happened, how it affected you and what you want to happen next.</p></div><a href="/member/account/">My Data</a></section>' +
      '<form class="member-form-panel story-form" data-story-form data-story-id="' + esc(editing.id || '') + '">' +
      '<div class="form-group"><label id="story-body-label">Your experience</label><div class="story-editor" data-story-editor-shell><div class="story-editor__toolbar" role="toolbar" aria-label="Text formatting"><button type="button" data-editor-command="bold" aria-label="Bold"><strong>B</strong></button><button type="button" data-editor-command="italic" aria-label="Italic"><em>I</em></button><button type="button" data-editor-command="insertUnorderedList" aria-label="Bulleted list">• List</button><button type="button" data-editor-command="insertOrderedList" aria-label="Numbered list">1. List</button><button type="button" data-editor-command="formatBlock" data-editor-value="blockquote" aria-label="Blockquote">“ Quote</button></div><div class="story-editor__surface" data-story-editor contenteditable="true" role="textbox" aria-multiline="true" aria-labelledby="story-body-label" data-placeholder="What happened? How did it affect you? What response did you receive?">' + markdownToEditorHTML(editing.bodyMarkdown || '') + '</div><div class="story-editor__footer"><span>Bold, italics, lists, quotes and your uploaded media</span><span data-story-character-count>0 / 20,000</span></div></div></div>' +
      '<div class="form-group"><label>Photos and short videos <span class="form-hint">(optional)</span></label><div class="story-media-picker" data-story-dropzone><label class="story-media-picker__button" for="story-media">Choose photos or videos<span>Use your photo library, camera roll or files</span></label><input class="visually-hidden" id="story-media" name="media" type="file" accept="image/jpeg,image/png,image/webp,video/mp4,video/quicktime,video/webm" multiple><p>Up to 5 photos and 2 videos; 500 MB total. You can select several at once.</p><p class="form-hint">On Mac Photos, use <strong>File → Export → Export 1 Photo</strong> and choose JPEG or PNG. HEIC files need exporting first.</p></div>' + (existingMedia ? '<p class="form-hint">Uploaded photos can be inserted into your experience. Select the image itself to open the full version.</p><ul class="story-existing-media">' + existingMedia + '</ul>' : '') + '<div class="story-media-previews" data-story-media-previews></div><p class="form-hint" data-story-media-status role="status"></p></div>' +
      '<div class="story-publication-settings"><p class="form-hint">Published as <strong>' + esc(state.displayName) + '</strong>. <a href="/member/account/">Change your display name in My Data</a>. Your contact details remain private.</p><fieldset><legend>Permissions</legend><label class="check-label"><input name="websiteConsent" type="checkbox" required' + checked(consent.website) + '> Publish this experience and selected media on this website.</label><label class="check-label"><input name="thirdPartyConsent" type="checkbox"' + checked(consent.thirdParty) + '> It may also be used in social media, press and publisher materials.</label><label class="check-label"><input name="copyrightConfirmed" type="checkbox"' + checked(consent.copyright) + '> I own, or have permission to supply, uploaded media.</label><label class="check-label"><input name="truthfulConfirmed" type="checkbox" required' + checked(consent.truthful) + '> This is true to the best of my knowledge.</label></fieldset></div>' +
      '<div class="story-vehicle-choice">' + vehicleControl + '</div>' +
      '<details class="story-evidence-picker"' + ((evidence.serviceEventIds || []).length || (evidence.sohReadingIds || []).length ? ' open' : '') + '><summary>Connect records from My Data <span>optional</span></summary><p>Linked records help turn individual experiences into evidence. <a href="/member/dashboard/">Add or check your records</a>.</p><div data-story-evidence><p class="form-hint">' + (selectedVehicle ? 'Loading records…' : 'Choose a car to see its records.') + '</p></div></details>' +
      '<div class="cluster story-form__actions"><button class="btn btn--primary" type="submit">Submit for review</button><button class="btn btn--secondary" type="button" data-save-story>Save draft</button>' + (editing.id ? '<button class="btn btn--secondary" type="button" data-cancel-edit>Cancel</button>' : '') + '</div><p data-story-status role="status" aria-live="polite"></p></form>' +
      (submissions ? '<section class="dashboard-panel story-submissions"><h2>Your experiences</h2><ul class="story-submission-list">' + submissions + '</ul></section>' : '');

    bindEditor();
    bindMedia();
    var vehicleSelect = root.querySelector('[name="vehicleId"]');
    if (vehicleSelect && vehicleSelect.tagName === 'SELECT') vehicleSelect.addEventListener('change', function () { renderEvidence(vehicleSelect.value, [], []); });
    if (selectedVehicle) renderEvidence(selectedVehicle, evidence.serviceEventIds || [], evidence.sohReadingIds || []);
    root.querySelector('[data-story-form]').addEventListener('submit', function (event) { event.preventDefault(); save(true); });
    root.querySelector('[data-save-story]').addEventListener('click', function () { save(false); });
    var cancel = root.querySelector('[data-cancel-edit]');
    if (cancel) cancel.addEventListener('click', function () { render(); });
    root.querySelectorAll('[data-edit-story]').forEach(function (button) {
      button.addEventListener('click', function () {
        var story = (state.stories || []).find(function (item) { return item.id === button.dataset.editStory; });
        if (story) { render(story); root.querySelector('[data-story-editor]').focus(); window.scrollTo({ top: root.offsetTop - 24, behavior: 'smooth' }); }
      });
    });
    root.querySelectorAll('[data-withdraw-story]').forEach(function (button) {
      button.addEventListener('click', function () {
        button.disabled = true;
        api('/api/member/story-withdraw', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id: button.dataset.withdrawStory }) }).then(load).catch(function (error) { button.disabled = false; button.insertAdjacentText('afterend', ' ' + error.message); });
      });
    });
  }

  function bindEditor() {
    var editor = root.querySelector('[data-story-editor]');
    var count = root.querySelector('[data-story-character-count]');
    function updateCount() { count.textContent = editorMarkdown().length.toLocaleString('en-GB') + ' / 20,000'; }
    editor.addEventListener('input', updateCount);
    editor.addEventListener('paste', function (event) {
      event.preventDefault();
      document.execCommand('insertText', false, event.clipboardData.getData('text/plain'));
    });
    root.querySelectorAll('[data-editor-command]').forEach(function (button) {
      button.addEventListener('mousedown', function (event) { event.preventDefault(); });
      button.addEventListener('click', function () { editor.focus(); document.execCommand(button.dataset.editorCommand, false, button.dataset.editorValue || null); updateCount(); });
    });
    root.querySelectorAll('[data-insert-story-media]').forEach(function (button) {
      button.addEventListener('mousedown', function (event) { event.preventDefault(); });
      button.addEventListener('click', function () {
        var item = editorMedia.find(function (media) { return media.id === button.dataset.insertStoryMedia; });
        if (!item || !item.url) return;
        editor.focus();
        var html = item.contentType.indexOf('video/') === 0
          ? '<p><video src="' + esc(item.url) + '" controls preload="metadata" data-story-media-id="' + esc(item.id) + '"></video></p>'
          : '<p><a class="story-inline-image" href="' + esc(item.url) + '" target="_blank" rel="noopener noreferrer"><img src="' + esc(item.url) + '" alt="' + esc(item.name) + '" data-story-media-id="' + esc(item.id) + '"></a></p>';
        document.execCommand('insertHTML', false, html);
        updateCount();
      });
    });
    updateCount();
  }

  function bindMedia() {
    var input = root.querySelector('#story-media');
    var dropzone = root.querySelector('[data-story-dropzone]');
    input.addEventListener('change', function () { addFiles(Array.from(input.files || [])); input.value = ''; });
    ['dragenter', 'dragover'].forEach(function (name) { dropzone.addEventListener(name, function (event) { event.preventDefault(); dropzone.classList.add('is-dragging'); }); });
    ['dragleave', 'drop'].forEach(function (name) { dropzone.addEventListener(name, function (event) { event.preventDefault(); dropzone.classList.remove('is-dragging'); }); });
    dropzone.addEventListener('drop', function (event) { addFiles(Array.from(event.dataTransfer.files || [])); });
  }

  function addFiles(files) {
    var status = root.querySelector('[data-story-media-status]');
    status.textContent = 'Preparing photos…';
    Promise.all(files.map(prepareImage)).then(function (prepared) { addPreparedFiles(prepared, status); });
  }

  function prepareImage(file) {
    if (!file || file.type.indexOf('image/') !== 0 || file.size <= imageResizeBytes || !window.createImageBitmap) return Promise.resolve(file);
    return window.createImageBitmap(file).then(function (bitmap) {
      var scale = Math.min(1, imageResizePixels / Math.max(bitmap.width, bitmap.height));
      if (scale === 1) { bitmap.close(); return file; }
      var canvas = document.createElement('canvas');
      canvas.width = Math.max(1, Math.round(bitmap.width * scale));
      canvas.height = Math.max(1, Math.round(bitmap.height * scale));
      canvas.getContext('2d').drawImage(bitmap, 0, 0, canvas.width, canvas.height);
      bitmap.close();
      return new Promise(function (resolve) {
        canvas.toBlob(function (blob) {
          resolve(blob ? new File([blob], file.name.replace(/\.[^.]+$/, '') + '.jpg', { type: 'image/jpeg', lastModified: file.lastModified }) : file);
        }, 'image/jpeg', 0.88);
      });
    }).catch(function () { return file; });
  }

  function addPreparedFiles(files, status) {
    var next = selectedFiles.slice();
    files.forEach(function (file) {
      if (!next.some(function (item) { return item.name === file.name && item.size === file.size && item.lastModified === file.lastModified; })) next.push(file);
    });
    var images = next.filter(function (file) { return file.type.indexOf('image/') === 0; }).length;
    var videos = next.filter(function (file) { return file.type.indexOf('video/') === 0; }).length;
    var bytes = next.reduce(function (total, file) { return total + file.size; }, 0);
    var unsupported = next.find(function (file) { return mediaTypes.indexOf(file.type) === -1; });
    if (unsupported) { status.textContent = unsupported.name + ' is not a supported photo or video format.'; return; }
    if (existingMediaUsage.images + images > state.limits.images || existingMediaUsage.videos + videos > state.limits.videos || existingMediaUsage.bytes + bytes > state.limits.totalBytes) { status.textContent = 'This story can contain no more than 5 photos and 2 videos, totalling 500 MB.'; return; }
    selectedFiles = next;
    status.textContent = selectedFiles.length ? selectedFiles.length + (selectedFiles.length === 1 ? ' item selected.' : ' items selected.') : '';
    renderMediaPreviews();
  }

  function renderMediaPreviews() {
    previewURLs.forEach(function (url) { URL.revokeObjectURL(url); });
    previewURLs = [];
    var previews = root.querySelector('[data-story-media-previews]');
    previews.innerHTML = selectedFiles.map(function (file, index) {
      var url = URL.createObjectURL(file);
      previewURLs.push(url);
      var image = file.type.indexOf('image/') === 0;
      var preview = image ? '<img src="' + esc(url) + '" alt="">' : '<video src="' + esc(url) + '" muted preload="metadata"></video>';
      return '<figure>' + preview + '<figcaption><span title="' + esc(file.name) + '">' + esc(file.name) + '</span><button type="button" data-insert-pending-media="' + index + '">Insert ' + (image ? 'image' : 'video') + '</button><button type="button" data-remove-media="' + index + '" aria-label="Remove ' + esc(file.name) + '">Remove</button></figcaption></figure>';
    }).join('');
    previews.querySelectorAll('[data-remove-media]').forEach(function (button) {
      button.addEventListener('click', function () { selectedFiles.splice(Number(button.dataset.removeMedia), 1); renderMediaPreviews(); root.querySelector('[data-story-media-status]').textContent = selectedFiles.length ? selectedFiles.length + ' selected.' : ''; });
    });
    previews.querySelectorAll('[data-insert-pending-media]').forEach(function (button) {
      button.addEventListener('click', function () {
        var index = Number(button.dataset.insertPendingMedia);
        var file = selectedFiles[index];
        var editor = root.querySelector('[data-story-editor]');
        if (!file || !editor) return;
        editor.focus();
        var url = previewURLs[index];
        var html = file.type.indexOf('video/') === 0
          ? '<p><video src="' + esc(url) + '" controls preload="metadata" data-story-pending-index="' + index + '"></video></p>'
          : '<p><a class="story-inline-image" href="' + esc(url) + '" target="_blank" rel="noopener noreferrer"><img src="' + esc(url) + '" alt="' + esc(file.name) + '" data-story-pending-index="' + index + '"></a></p>';
        document.execCommand('insertHTML', false, html);
        root.querySelector('[data-story-character-count]').textContent = editorMarkdown().length.toLocaleString('en-GB') + ' / 20,000';
        button.textContent = 'Inserted; upload to save';
      });
    });
  }

  function attachUploadedMedia(index, media) {
    var editor = root.querySelector('[data-story-editor]');
    if (!editor || !media) return;
    editor.querySelectorAll('[data-story-pending-index="' + index + '"]').forEach(function (node) {
      node.setAttribute('data-story-media-id', media.id);
      node.removeAttribute('data-story-pending-index');
    });
  }

  function renderEvidence(vehicleId, selectedService, selectedSOH) {
    var service = (state.serviceEvents || []).filter(function (item) { return item.vehicleId === vehicleId; });
    var soh = (state.sohReadings || []).filter(function (item) { return item.vehicleId === vehicleId; });
    var box = root.querySelector('[data-story-evidence]');
    function checks(items, name, selected, labeler) {
      return items.map(function (item) { return '<label class="check-label"><input type="checkbox" name="' + name + '" value="' + esc(item.id) + '"' + checked(selected.indexOf(item.id) !== -1) + '> ' + esc(labeler(item)) + '</label>'; }).join('');
    }
    var serviceHTML = checks(service, 'serviceEventIds', selectedService || [], function (item) { return (item.occurredAt || 'No date') + ' · ' + (item.title || item.eventType || 'Record'); });
    var sohHTML = checks(soh, 'sohReadingIds', selectedSOH || [], function (item) { var battery = item.battery || {}; return (battery.measuredAt || 'No date') + ' · ' + (battery.stateOfHealth == null ? 'SoH reading' : battery.stateOfHealth + '% SoH'); });
    box.innerHTML = '<h3>Service, fault and JLR contacts</h3>' + (serviceHTML || '<p class="form-hint">No records yet.</p>') + '<h3>Battery SoH readings</h3>' + (sohHTML || '<p class="form-hint">No readings yet.</p>');
  }

  function payload(form, submit) {
    var data = new FormData(form);
    return { id: form.dataset.storyId, bodyMarkdown: editorMarkdown(), vehicleId: data.get('vehicleId'), serviceEventIds: data.getAll('serviceEventIds'), sohReadingIds: data.getAll('sohReadingIds'), websiteConsent: data.has('websiteConsent'), thirdPartyConsent: data.has('thirdPartyConsent'), copyrightConfirmed: data.has('copyrightConfirmed'), truthfulConfirmed: data.has('truthfulConfirmed'), submit: submit };
  }

  function upload(storyId, file) {
    return api('/api/member/story-upload', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ storyId: storyId, name: file.name, contentType: file.type, size: file.size }) }).then(function (reservation) {
      return fetch(reservation.uploadUrl, { method: 'PUT', headers: { 'Content-Type': reservation.contentType }, body: file }).catch(function (error) {
        error.traceId = reservation.traceId || '';
        throw error;
      }).then(function (response) {
        if (!response.ok) { var uploadError = new Error('Could not upload ' + file.name); uploadError.traceId = reservation.traceId || ''; throw uploadError; }
        return api('/api/member/story-upload-complete', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ storyId: storyId, mediaId: reservation.mediaId }) }).then(function () { return { id: reservation.mediaId }; });
      });
    });
  }

  function save(submit) {
    var form = root.querySelector('[data-story-form]');
    var status = root.querySelector('[data-story-status]');
    var body = editorMarkdown();
    if (!body || body.length > 20000 || (submit && body.length < 80)) {
      status.textContent = submit && body.length < 80 ? 'Please add a little more detail before submitting.' : 'Your experience must be between 1 and 20,000 characters.';
      root.querySelector('[data-story-editor]').focus();
      return;
    }
    if (submit && !form.reportValidity()) return;
    var data = payload(form, false);
    var files = selectedFiles.slice();
    status.textContent = 'Saving…';
    api('/api/member/stories', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) }).then(function (result) {
      return files.reduce(function (chain, file, index) { return chain.then(function () { status.textContent = 'Uploading ' + file.name + '…'; return upload(result.story.id, file).then(function (media) { attachUploadedMedia(index, media); }); }); }, Promise.resolve()).then(function () {
        if (!files.length && !submit) return result;
        data.id = result.story.id;
        data.bodyMarkdown = editorMarkdown();
        data.submit = submit;
        return api('/api/member/stories', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(data) });
      });
    }).then(function () { status.textContent = submit ? 'Submitted for review.' : 'Draft saved.'; return load(); }).catch(function (error) { status.textContent = displayError(error); });
  }

  document.addEventListener('member:data', load);
  document.addEventListener('identity:login', load);
  if (window.ipaceIdentityUser) load();
})();
