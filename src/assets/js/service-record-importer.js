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
  function field(form, grid, labelText, name, type, value, wide, options) {
    if (!form) return null;
    var group = document.createElement('div'); group.className = 'form-group' + (wide ? ' member-form-grid__wide' : ''); var label = document.createElement('label'); label.textContent = labelText; var control = document.createElement(options ? 'select' : type === 'textarea' ? 'textarea' : 'input'); control.name = name; control.required = name === 'vehicleId' || name === 'occurredAt' || name === 'title'; if (!options && type !== 'textarea') control.type = type; if (type === 'textarea') { control.rows = 5; control.maxLength = 4000; } if (name === 'title') control.maxLength = 160; if (name === 'serviceProviderName') control.maxLength = 180; if (options) options.forEach(function (option) { var item = document.createElement('option'); item.value = option[0]; item.textContent = option[1]; control.appendChild(item); }); control.value = value == null ? '' : value; label.appendChild(control); group.appendChild(label); grid.appendChild(group); return control;
  }
  function recordCard(record, file, method) {
    var card = document.createElement('article'); card.className = 'service-import-card'; card.dataset.importCard = ''; var source = document.createElement('div'); source.className = 'service-import-card__source'; var heading = document.createElement('h3'); heading.textContent = file.name; var hint = document.createElement('p'); hint.className = 'form-hint'; hint.textContent = method + ' · fingerprint ' + record.hash.slice(0, 12) + '…'; var preview = document.createElement(file.type === 'application/pdf' ? 'iframe' : 'img'); preview.src = URL.createObjectURL(file); if (preview.tagName === 'IFRAME') preview.title = 'Local preview: ' + file.name; else preview.alt = 'Local preview of ' + file.name; source.append(heading, hint, preview);
    var form = document.createElement('form'); form.className = 'service-import-card__form'; form.dataset.importRecord = ''; var grid = document.createElement('div'); grid.className = 'member-form-grid'; var vehicles = [['', 'Choose vehicle']].concat((memberData.vehicleRecords || []).map(function (item) { var vehicle = item.vehicle || {}; return [item.id, vehicle.registration || 'Vehicle ' + item.id.slice(-6)]; })); field(form, grid, 'Vehicle', 'vehicleId', 'select', '', false, vehicles); field(form, grid, 'Date', 'occurredAt', 'date', record.occurredAt); field(form, grid, 'Type', 'eventType', 'select', record.eventType, false, [['service', 'Service'], ['fault', 'Fault'], ['repair', 'Repair'], ['recall', 'Recall'], ['inspection', 'Inspection'], ['other', 'Other']]); field(form, grid, 'Status', 'status', 'select', record.status, false, [['open', 'Open'], ['monitoring', 'Monitoring'], ['resolved', 'Resolved'], ['completed', 'Completed']]); var mileage = field(form, grid, 'Mileage', 'mileage', 'number', record.mileage); mileage.min = 0; mileage.max = 500000; field(form, grid, 'Provider', 'serviceProviderName', 'text', record.serviceProviderName); field(form, grid, 'Summary', 'title', 'text', record.title, true); field(form, grid, 'Details', 'description', 'textarea', record.description, true); var warranty = document.createElement('input'); warranty.type = 'hidden'; warranty.name = 'warrantyCover'; warranty.value = record.warrantyCover;
    var actions = document.createElement('div'); actions.className = 'cluster'; var submit = document.createElement('button'); submit.className = 'btn btn--primary'; submit.type = 'submit'; submit.textContent = 'Submit reviewed record'; var discard = document.createElement('button'); discard.className = 'btn btn--secondary'; discard.type = 'button'; discard.dataset.removeImport = ''; discard.textContent = 'Discard'; actions.append(submit, discard); var message = document.createElement('p'); message.className = 'form-hint'; message.dataset.recordStatus = ''; message.setAttribute('role', 'status'); message.setAttribute('aria-live', 'polite'); form.append(grid, warranty, actions, message); card.append(source, form); return card;
  }
  function selectValue(form, name, value) { if (value) form.elements[name].value = value; }
  function addRecord(file, text, hash, method) { var record = guess(file, text, hash); record.hash = hash; var card = recordCard(record, file, method); results.appendChild(card); var form = card.querySelector('form'); selectValue(form, 'eventType', record.eventType); selectValue(form, 'status', record.status); }
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
