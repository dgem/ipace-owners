(function () {
  'use strict';

  var root = document.querySelector('[data-service-importer]');
  if (!root) return;
  var fileInput = root.querySelector('[data-import-files]');
  var status = root.querySelector('[data-import-status]');
  var results = root.querySelector('[data-import-results]');
  var memberData = null;
  var hashes = {};
  var ocrWorker = null;

  function escapeHtml(value) { return String(value == null ? '' : value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;'); }
  function dateFrom(value) {
    var match = String(value).match(/(?:^|[^0-9])(0?[1-9]|[12][0-9]|3[01])[./-](0?[1-9]|1[0-2])[./-]([0-9]{2,4})(?:[^0-9]|$)/);
    if (!match) return '';
    var year = Number(match[3]); if (year < 100) year += year > 70 ? 1900 : 2000;
    return year + '-' + String(match[2]).padStart(2, '0') + '-' + String(match[1]).padStart(2, '0');
  }
  function extractMileage(text) {
    var match = text.match(/(?:mileage|odometer|miles?)\D{0,20}([0-9]{1,3}(?:[, ]?[0-9]{3})*)/i);
    return match ? Number(match[1].replace(/[, ]/g, '')) : '';
  }
  function guess(file, text, hash) {
    var source = file.name.replace(/\.[^.]+$/, '');
    var haystack = (source + '\n' + text).toLowerCase();
    var type = /recall|campaign/.test(haystack) ? 'recall' : /warranty|repair|rectif/.test(haystack) ? 'repair' : /fault|issue|concern|error/.test(haystack) ? 'fault' : /inspect|vhc/.test(haystack) ? 'inspection' : 'service';
    var warranty = /warranty/.test(haystack);
    return { occurredAt: dateFrom(source) || dateFrom(text), mileage: extractMileage(text), eventType: type, status: warranty || /invoice|cash/.test(haystack) ? 'completed' : 'resolved', serviceProviderName: /hendy/i.test(haystack) ? 'Hendy Jaguar' : '', warrantyCover: warranty ? 'manufacturer' : '', title: source.slice(0, 160), description: 'Source retained locally: ' + file.name + '\nSHA-256: ' + hash + '\n\nCheck this draft against the original document before submitting.' };
  }
  function hashFile(file) { return file.arrayBuffer().then(function (buffer) { return crypto.subtle.digest('SHA-256', buffer); }).then(function (digest) { return Array.from(new Uint8Array(digest)).map(function (n) { return n.toString(16).padStart(2, '0'); }).join(''); }); }
  function pdfText(file) {
    return import('/assets/vendor/pdfjs/pdf.min.mjs').then(function (pdfjs) {
      pdfjs.GlobalWorkerOptions.workerSrc = '/assets/vendor/pdfjs/pdf.worker.min.mjs';
      return file.arrayBuffer().then(function (buffer) { return pdfjs.getDocument({ data: new Uint8Array(buffer) }).promise; });
    }).then(function (documentPdf) {
      var pages = []; for (var index = 1; index <= documentPdf.numPages; index += 1) pages.push(documentPdf.getPage(index).then(function (page) { return page.getTextContent().then(function (content) { return content.items.map(function (item) { return item.str; }).join(' '); }); }));
      return Promise.all(pages).then(function (items) { return items.join('\n'); });
    });
  }
  function loadOCR() {
    if (ocrWorker) return ocrWorker;
    ocrWorker = new Promise(function (resolve, reject) {
      var script = document.createElement('script'); script.src = '/assets/vendor/tesseract/tesseract.min.js'; script.onload = resolve; script.onerror = reject; document.head.appendChild(script);
    }).then(function () { return window.Tesseract.createWorker('eng', 1, { workerPath: '/assets/vendor/tesseract/worker.min.js', corePath: '/assets/vendor/tesseract-core', langPath: '/assets/vendor/tessdata/4.0.0', logger: function () {} }); });
    return ocrWorker;
  }
  function firstPDFPage(file) {
    return import('/assets/vendor/pdfjs/pdf.min.mjs').then(function (pdfjs) { pdfjs.GlobalWorkerOptions.workerSrc = '/assets/vendor/pdfjs/pdf.worker.min.mjs'; return file.arrayBuffer().then(function (buffer) { return pdfjs.getDocument({ data: new Uint8Array(buffer) }).promise; }); }).then(function (pdf) { return pdf.getPage(1); }).then(function (page) {
      var viewport = page.getViewport({ scale: 1.75 }); var canvas = document.createElement('canvas'); canvas.width = viewport.width; canvas.height = viewport.height; return page.render({ canvasContext: canvas.getContext('2d'), viewport: viewport }).promise.then(function () { return canvas.toDataURL('image/png'); });
    });
  }
  function ocrFile(file) {
    var input = file.type === 'application/pdf' ? firstPDFPage(file) : Promise.resolve(URL.createObjectURL(file));
    return input.then(function (image) { return loadOCR().then(function (worker) { return worker.recognize(image).then(function (result) { if (image.indexOf('blob:') === 0) URL.revokeObjectURL(image); return result.data.text; }); }); });
  }
  function vehicleOptions() { return (memberData && memberData.vehicleRecords || []).map(function (record) { var vehicle = record.vehicle || {}; return '<option value="' + escapeHtml(record.id) + '">' + escapeHtml(vehicle.registration || 'Vehicle ' + record.id.slice(-6)) + '</option>'; }).join(''); }
  function recordMarkup(record, file, method) {
    var preview = URL.createObjectURL(file); var options = vehicleOptions();
    return '<article class="service-import-card" data-import-card><div class="service-import-card__source"><h3>' + escapeHtml(file.name) + '</h3><p class="form-hint">' + escapeHtml(method) + ' · fingerprint ' + escapeHtml(record.hash.slice(0, 12)) + '…</p>' + (file.type === 'application/pdf' ? '<iframe title="Local preview: ' + escapeHtml(file.name) + '" src="' + preview + '"></iframe>' : '<img alt="Local preview of ' + escapeHtml(file.name) + '" src="' + preview + '">') + '</div><form class="service-import-card__form" data-import-record><div class="member-form-grid"><div class="form-group"><label>Vehicle<select name="vehicleId" required><option value="">Choose vehicle</option>' + options + '</select></label></div><div class="form-group"><label>Date<input name="occurredAt" type="date" value="' + escapeHtml(record.occurredAt) + '" required></label></div><div class="form-group"><label>Type<select name="eventType"><option value="service">Service</option><option value="fault">Fault</option><option value="repair">Repair</option><option value="recall">Recall</option><option value="inspection">Inspection</option><option value="other">Other</option></select></label></div><div class="form-group"><label>Status<select name="status"><option value="open">Open</option><option value="monitoring">Monitoring</option><option value="resolved">Resolved</option><option value="completed">Completed</option></select></label></div><div class="form-group"><label>Mileage<input name="mileage" type="number" min="0" max="500000" value="' + escapeHtml(record.mileage) + '"></label></div><div class="form-group"><label>Provider<input name="serviceProviderName" maxlength="180" value="' + escapeHtml(record.serviceProviderName) + '"></label></div><div class="form-group member-form-grid__wide"><label>Summary<input name="title" maxlength="160" required value="' + escapeHtml(record.title) + '"></label></div><div class="form-group member-form-grid__wide"><label>Details<textarea name="description" rows="5" maxlength="4000">' + escapeHtml(record.description) + '</textarea></label></div></div><input name="warrantyCover" type="hidden" value="' + escapeHtml(record.warrantyCover) + '"><div class="cluster"><button class="btn btn--primary" type="submit">Submit reviewed record</button><button class="btn btn--secondary" type="button" data-remove-import>Discard</button></div><p class="form-hint" data-record-status role="status" aria-live="polite"></p></form></article>';
  }
  function selectValue(form, name, value) { if (value) form.elements[name].value = value; }
  function addRecord(file, text, hash, method) { var record = guess(file, text, hash); record.hash = hash; results.insertAdjacentHTML('beforeend', recordMarkup(record, file, method)); var form = results.lastElementChild.querySelector('form'); selectValue(form, 'eventType', record.eventType); selectValue(form, 'status', record.status); }
  fileInput.addEventListener('change', function () {
    var files = Array.from(fileInput.files || []); if (!files.length) return; if (!memberData || !(memberData.vehicleRecords || []).length) { status.textContent = 'Add a vehicle before importing service records.'; return; }
    status.textContent = 'Reading ' + files.length + ' local file' + (files.length === 1 ? '' : 's') + '…';
    files.reduce(function (chain, file) { return chain.then(function () { return hashFile(file).then(function (hash) { if (hashes[hash]) { status.textContent = 'Skipped duplicate: ' + file.name; return; } hashes[hash] = true; return (file.type === 'application/pdf' ? pdfText(file) : Promise.resolve('')).then(function (text) { if (text.trim().length > 40) { addRecord(file, text, hash, 'Embedded PDF text'); return; } status.textContent = 'Running local OCR for ' + file.name + '…'; return ocrFile(file).then(function (ocrText) { addRecord(file, ocrText, hash, 'Local OCR'); }); }); }); }); }, Promise.resolve()).then(function () { status.textContent = 'Ready for review. No source file has been uploaded.'; }).catch(function (error) { status.textContent = 'Could not read one of the files: ' + (error.message || 'unknown error'); });
  });
  results.addEventListener('click', function (event) { var remove = event.target.closest('[data-remove-import]'); if (remove) remove.closest('[data-import-card]').remove(); });
  results.addEventListener('submit', function (event) {
    var form = event.target.closest('[data-import-record]'); if (!form) return; event.preventDefault(); var message = form.querySelector('[data-record-status]'); var payload = Object.fromEntries(new FormData(form).entries()); if (payload.mileage === '') delete payload.mileage; message.textContent = 'Submitting reviewed record…'; form.querySelector('button[type="submit"]').disabled = true;
    Promise.resolve(window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : '').then(function (token) { return fetch('/api/upsert-service-event', { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token }, body: JSON.stringify(payload) }); }).then(function (response) { return response.json().then(function (data) { if (!response.ok || !data.ok) throw new Error(data.error || 'Could not save record'); }); }).then(function () { message.textContent = 'Saved. The original document remains only on this device.'; form.querySelector('button[type="submit"]').remove(); if (window.ipaceRefreshMemberData) window.ipaceRefreshMemberData(); }).catch(function (error) { message.textContent = error.message || 'Could not save record.'; form.querySelector('button[type="submit"]').disabled = false; });
  });
  document.addEventListener('member:data', function (event) { memberData = event.detail && event.detail.data; });
  var authContainer = root.closest('[data-auth-container]');
  if (authContainer && authContainer.dataset.memberData) {
    try { memberData = JSON.parse(authContainer.dataset.memberData); } catch { memberData = null; }
  }
}());
