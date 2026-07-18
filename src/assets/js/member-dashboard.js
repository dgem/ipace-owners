/**
 * member-dashboard.js — One-vehicle-at-a-time member workspace.
 */
(function () {
  'use strict';

  var workspace = document.querySelector('[data-vehicle-workspace]');
  if (!workspace) return;

  var activeVehicleId = '';
  var memberData = null;

  function escapeHtml(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function formatDate(value) {
    if (!value) return '';
    var date = new Date(value + (String(value).length === 10 ? 'T12:00:00' : ''));
    return Number.isNaN(date.getTime()) ? '' : date.toLocaleDateString('en-GB');
  }

  function todayString() {
    var date = new Date();
    return date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0');
  }

  function notFutureDateAttributes(errorId) {
    return ' data-not-future max="' + todayString() + '" aria-describedby="' + errorId + '"';
  }

  function vehicleName(record) {
    var vehicle = record.vehicle || {};
    return vehicle.registration || ('Vehicle ' + String(record.id || '').slice(-6));
  }

  function sourceLabel(value) {
    return ({
      'dealer-report': 'Dealer report',
      'diagnostic-app': 'Diagnostic app / OBD',
      'service-paperwork': 'Service paperwork',
      'jlr-communication': 'JLR communication',
      'estimate-unsure': 'Estimate / unsure',
    })[value] || String(value || '').replace(/-/g, ' ');
  }

  function eventTypeLabel(value) {
    return ({ service: 'Service', fault: 'Fault', repair: 'Repair', recall: 'Recall', inspection: 'Inspection', other: 'Other' })[value] || value;
  }

  function statusLabel(value) {
    return ({ open: 'Open', monitoring: 'Monitoring', resolved: 'Resolved', completed: 'Completed' })[value] || value;
  }

  function supportLabel(value) {
    return ({
      yes: 'Yes',
      no: 'No',
      'not-needed': 'Not needed',
      unsure: 'Unsure',
      partly: 'Partly',
      manufacturer: 'Manufacturer warranty',
      'battery-warranty': '8-year battery warranty',
      'extended-manufacturer': 'Extended manufacturer warranty',
      'third-party': 'Third-party warranty',
      none: 'None',
      'initially-refused': 'Initially refused',
      'partially-accepted': 'Partially accepted',
      'still-disputed': 'Still disputed',
      'resolved-after-escalation': 'Resolved after escalation',
    })[value] || value;
  }

  function readingsFor(vehicleId) {
    return (memberData.batteryReadings || []).filter(function (reading) {
      return reading.vehicleId === vehicleId && reading.battery && reading.battery.stateOfHealth != null;
    }).sort(function (a, b) {
      return String(a.battery.measuredAt || '').localeCompare(String(b.battery.measuredAt || ''));
    });
  }

  function eventsFor(vehicleId) {
    return (memberData.serviceEvents || []).filter(function (item) {
      return item.vehicleId === vehicleId;
    }).sort(function (a, b) {
      return String(b.occurredAt || '').localeCompare(String(a.occurredAt || ''));
    });
  }

  function chartMarkup(readings) {
    if (!readings.length) {
      return '<div class="vehicle-empty-state"><p>No State of Health readings recorded yet.</p></div>';
    }

    var width = 900;
    var height = 280;
    var left = 58;
    var right = 22;
    var top = 22;
    var bottom = 46;
    var values = readings.map(function (reading) { return Number(reading.battery.stateOfHealth); });
    var minimum = Math.max(0, Math.floor((Math.min.apply(null, values) - 5) / 10) * 10);
    var maximum = Math.min(100, Math.ceil((Math.max.apply(null, values) + 5) / 10) * 10);
    if (minimum === maximum) {
      minimum = Math.max(0, minimum - 10);
      maximum = Math.min(100, maximum + 10);
    }
    var plotWidth = width - left - right;
    var plotHeight = height - top - bottom;
    var points = readings.map(function (reading, index) {
      var x = readings.length === 1 ? left + plotWidth / 2 : left + (index / (readings.length - 1)) * plotWidth;
      var y = top + ((maximum - Number(reading.battery.stateOfHealth)) / (maximum - minimum)) * plotHeight;
      return { x: x, y: y, reading: reading };
    });
    var description = readings.map(function (reading) {
      return reading.battery.stateOfHealth + '% on ' + formatDate(reading.battery.measuredAt);
    }).join(', ');
    var grid = '';
    for (var i = 0; i <= 4; i += 1) {
      var y = top + (i / 4) * plotHeight;
      var label = Math.round(maximum - (i / 4) * (maximum - minimum));
      grid += '<line x1="' + left + '" y1="' + y + '" x2="' + (width - right) + '" y2="' + y + '" class="soh-chart__grid" />';
      grid += '<text x="' + (left - 10) + '" y="' + (y + 4) + '" text-anchor="end" class="soh-chart__label">' + label + '%</text>';
    }
    var circles = points.map(function (point) {
      return '<circle cx="' + point.x + '" cy="' + point.y + '" r="5" class="soh-chart__point"><title>' +
        escapeHtml(point.reading.battery.stateOfHealth + '% on ' + formatDate(point.reading.battery.measuredAt)) + '</title></circle>';
    }).join('');
    var dateLabels = '';
    if (points.length) {
      dateLabels += '<text x="' + points[0].x + '" y="' + (height - 14) + '" text-anchor="start" class="soh-chart__label">' + escapeHtml(formatDate(readings[0].battery.measuredAt)) + '</text>';
      if (points.length > 1) {
        dateLabels += '<text x="' + points[points.length - 1].x + '" y="' + (height - 14) + '" text-anchor="end" class="soh-chart__label">' + escapeHtml(formatDate(readings[readings.length - 1].battery.measuredAt)) + '</text>';
      }
    }
    return '<div class="soh-chart"><svg viewBox="0 0 ' + width + ' ' + height + '" role="img" aria-label="State of Health history: ' + escapeHtml(description) + '">' +
      grid + '<polyline points="' + points.map(function (point) { return point.x + ',' + point.y; }).join(' ') + '" class="soh-chart__line" />' + circles + dateLabels + '</svg></div>';
  }

  function readingsMarkup(readings) {
    if (!readings.length) return '';
    return '<div class="reading-table-wrap"><table class="reading-table"><thead><tr><th>Date</th><th>SoH</th><th>Mileage</th><th>Source</th></tr></thead><tbody>' +
      readings.slice().reverse().map(function (reading) {
        var battery = reading.battery || {};
        return '<tr><td>' + escapeHtml(formatDate(battery.measuredAt)) + '</td><td><strong>' + escapeHtml(battery.stateOfHealth) + '%</strong></td><td>' +
          (battery.mileageAtMeasurement == null ? 'Not recorded' : Number(battery.mileageAtMeasurement).toLocaleString() + ' miles') + '</td><td>' + escapeHtml(sourceLabel(battery.source)) + '</td></tr>';
      }).join('') + '</tbody></table></div>';
  }

  function sohFormMarkup(vehicleId) {
    var id = escapeHtml(vehicleId);
    return '<div class="member-form-panel" data-soh-panel hidden><form data-soh-update-form>' +
      '<input type="hidden" name="vehicleId" value="' + id + '"><div class="member-form-grid">' +
      '<div class="form-group"><label for="dashboard-soh">State of Health (%)</label><input id="dashboard-soh" name="soh" type="number" min="0" max="100" step="0.1" required></div>' +
      '<div class="form-group"><label for="dashboard-soh-date">Measurement date</label><input id="dashboard-soh-date" name="sohDate" type="date" required' + notFutureDateAttributes('dashboard-soh-date-error') + '><span class="form-error" id="dashboard-soh-date-error" role="alert" hidden>Measurement date cannot be in the future.</span></div>' +
      '<div class="form-group"><label for="dashboard-soh-mileage">Mileage at measurement</label><input id="dashboard-soh-mileage" name="sohMileage" type="number" min="0" max="500000"></div>' +
      '<div class="form-group"><label for="dashboard-soh-source">Measurement source</label><select id="dashboard-soh-source" name="sohSource" required><option value="">Select source</option><option value="dealer-report">Dealer report</option><option value="diagnostic-app">Diagnostic app / OBD</option><option value="service-paperwork">Service paperwork</option><option value="jlr-communication">JLR communication</option><option value="estimate-unsure">Estimate / unsure</option></select></div>' +
      '</div><div class="cluster"><button class="btn btn--primary" type="submit">Save reading</button><button class="btn btn--secondary" type="button" data-close-panel="soh">Cancel</button></div>' +
      '<p class="form-hint" data-soh-update-status role="status" aria-live="polite"></p></form></div>';
  }

  function serviceEventFormMarkup(vehicleId) {
    return '<div class="member-form-panel" data-event-panel hidden><form data-service-event-form>' +
      '<input type="hidden" name="id"><input type="hidden" name="vehicleId" value="' + escapeHtml(vehicleId) + '">' +
      '<h3 data-event-form-title>Add service event or fault</h3><div class="member-form-grid">' +
      '<div class="form-group"><label for="event-type">Record type</label><select id="event-type" name="eventType" required><option value="fault" selected>Fault</option><option value="service">Service</option><option value="repair">Repair</option><option value="recall">Recall</option><option value="inspection">Inspection</option><option value="other">Other</option></select></div>' +
      '<div class="form-group"><label for="event-date">Date</label><input id="event-date" name="occurredAt" type="date" required' + notFutureDateAttributes('event-date-error') + '><span class="form-error" id="event-date-error" role="alert" hidden>Event date cannot be in the future.</span></div>' +
      '<div class="form-group"><label for="event-mileage">Mileage</label><input id="event-mileage" name="mileage" type="number" min="0" max="500000"></div>' +
      '<div class="form-group"><label for="event-status">Status</label><select id="event-status" name="status" required><option value="open">Open</option><option value="monitoring">Monitoring</option><option value="resolved">Resolved</option><option value="completed">Completed</option></select></div>' +
      '<fieldset class="form-group member-form-grid__wide"><legend>Related campaigns or recalls</legend><div class="check-group check-group--inline">' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="H447"> H447</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="H570"> H570</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="H571"> H571</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="H572"> H572</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="other"> Other</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="unsure"> Unsure</label>' +
      '<label class="check-label"><input type="checkbox" name="campaigns" value="none"> None</label></div></fieldset>' +
      '<div class="form-group"><label for="event-final-fix">Final fix date</label><input id="event-final-fix" name="finalFixAt" type="date"' + notFutureDateAttributes('event-final-fix-error') + '><span class="form-error" id="event-final-fix-error" role="alert" hidden>Final fix date cannot be in the future.</span></div>' +
      '<div class="form-group"><label for="event-days-to-fix">Days from fault to final fix</label><input id="event-days-to-fix" name="daysToFinalFix" type="number" min="0" max="5000"></div>' +
      '<div class="form-group"><label for="event-courtesy-offered">Courtesy vehicle offered?</label><select id="event-courtesy-offered" name="courtesyVehicleOffered"><option value="">Select</option><option value="yes">Yes</option><option value="no">No</option><option value="not-needed">Not needed</option><option value="unsure">Unsure</option></select></div>' +
      '<div class="form-group"><label for="event-courtesy-provided">Courtesy vehicle provided?</label><select id="event-courtesy-provided" name="courtesyVehicleProvided"><option value="">Select</option><option value="yes">Yes</option><option value="no">No</option><option value="not-needed">Not needed</option><option value="unsure">Unsure</option></select></div>' +
      '<div class="form-group"><label for="event-parts-delay">Delay due to parts?</label><select id="event-parts-delay" name="partsDelay"><option value="">Select</option><option value="yes">Yes</option><option value="partly">Partly</option><option value="no">No</option><option value="unsure">Unsure</option></select></div>' +
      '<div class="form-group"><label for="event-warranty-cover">Warranty cover in place</label><select id="event-warranty-cover" name="warrantyCover"><option value="">Select</option><option value="manufacturer">Manufacturer warranty</option><option value="battery-warranty">8-year battery warranty</option><option value="extended-manufacturer">Extended manufacturer warranty</option><option value="third-party">Third-party warranty</option><option value="none">No warranty cover</option><option value="unsure">Unsure</option></select></div>' +
      '<div class="form-group"><label for="event-dispute-status">Responsibility or warranty dispute?</label><select id="event-dispute-status" name="disputeStatus"><option value="">Select</option><option value="none">No dispute</option><option value="initially-refused">Initially refused</option><option value="partially-accepted">Partially accepted only</option><option value="still-disputed">Still disputed</option><option value="resolved-after-escalation">Resolved after escalation</option><option value="unsure">Unsure</option></select></div>' +
      '<div class="form-group member-form-grid__wide"><label for="event-title">Summary</label><input id="event-title" name="title" type="text" maxlength="160" required></div>' +
      '<div class="form-group member-form-grid__wide"><label for="event-description">Details</label><textarea id="event-description" name="description" rows="4" maxlength="4000"></textarea></div>' +
      '</div><div class="cluster"><button class="btn btn--primary" type="submit">Save record</button><button class="btn btn--secondary" type="button" data-close-panel="event">Cancel</button></div>' +
      '<p class="form-hint" data-service-event-status role="status" aria-live="polite"></p></form></div>';
  }

  function serviceEventsMarkup(events) {
    if (!events.length) return '<div class="vehicle-empty-state"><p>No service events or faults recorded yet.</p></div>';
    return '<div class="service-event-list">' + events.map(function (item) {
      var meta = [formatDate(item.occurredAt)];
      if (item.mileage != null) meta.push(Number(item.mileage).toLocaleString() + ' miles');
      if (item.campaigns && item.campaigns.length) meta.push('Campaigns: ' + item.campaigns.join(', '));
      var support = [];
      if (item.finalFixAt) support.push('Final fix: ' + formatDate(item.finalFixAt));
      if (item.daysToFinalFix != null) support.push('Days to final fix: ' + Number(item.daysToFinalFix).toLocaleString());
      if (item.courtesyVehicleOffered) support.push('Courtesy offered: ' + supportLabel(item.courtesyVehicleOffered));
      if (item.courtesyVehicleProvided) support.push('Courtesy provided: ' + supportLabel(item.courtesyVehicleProvided));
      if (item.partsDelay) support.push('Parts delay: ' + supportLabel(item.partsDelay));
      if (item.warrantyCover) support.push('Warranty: ' + supportLabel(item.warrantyCover));
      if (item.disputeStatus) support.push('Dispute: ' + supportLabel(item.disputeStatus));
      return '<article class="service-event"><div class="service-event__main"><div class="cluster cluster--sm"><span class="badge">' + escapeHtml(eventTypeLabel(item.eventType)) + '</span><span class="badge badge--muted">' + escapeHtml(statusLabel(item.status)) + '</span></div>' +
        '<h3>' + escapeHtml(item.title) + '</h3><p class="service-event__meta">' + escapeHtml(meta.join(' · ')) + '</p>' +
        (support.length ? '<p class="service-event__meta">' + escapeHtml(support.join(' · ')) + '</p>' : '') +
        (item.description ? '<p>' + escapeHtml(item.description) + '</p>' : '') + '</div>' +
        '<button class="btn btn--sm btn--secondary" type="button" data-edit-event="' + escapeHtml(item.id) + '">Edit</button></article>';
    }).join('') + '</div>';
  }

  function renderActiveVehicle(record) {
    var target = workspace.querySelector('[data-active-vehicle]');
    var vehicle = record.vehicle || {};
    var readings = readingsFor(record.id);
    var events = eventsFor(record.id);
    var meta = [];
    if (vehicle.modelYear) meta.push('Model year ' + vehicle.modelYear);
    if (vehicle.country) meta.push(vehicle.country);
    if (vehicle.mileage != null) meta.push(Number(vehicle.mileage).toLocaleString() + ' miles');
    target.innerHTML = '<div class="vehicle-workspace-panel" role="tabpanel" tabindex="0" aria-labelledby="vehicle-tab-' + escapeHtml(record.id) + '">' +
      '<header class="vehicle-workspace-header"><div><p class="page-header__eyebrow">Selected vehicle</p><h2>' + escapeHtml(vehicleName(record)) + '</h2><p class="vehicle-workspace-meta">' + escapeHtml(meta.join(' · ')) + '</p></div></header>' +
      '<section class="vehicle-data-section" aria-labelledby="soh-history-title"><div class="vehicle-section-header"><div><h2 id="soh-history-title">State of Health history</h2><p>Track battery health readings over time.</p></div><button class="btn btn--primary" type="button" data-open-panel="soh">Add reading</button></div>' +
      sohFormMarkup(record.id) + chartMarkup(readings) + readingsMarkup(readings) + '</section>' +
      '<section class="vehicle-data-section" aria-labelledby="service-events-title"><div class="vehicle-section-header"><div><h2 id="service-events-title">Service events and faults</h2><p>Keep a dated record of servicing, recalls, faults and repairs.</p></div><button class="btn btn--primary" type="button" data-open-panel="event">Add record</button></div>' +
      serviceEventFormMarkup(record.id) + serviceEventsMarkup(events) + '</section></div>';
  }

  function render() {
    var loading = workspace.querySelector('[data-vehicle-workspace-loading]');
    var content = workspace.querySelector('[data-vehicle-workspace-content]');
    var tabs = workspace.querySelector('[data-vehicle-tabs]');
    var target = workspace.querySelector('[data-active-vehicle]');
    var vehicles = memberData.vehicleRecords || [];
    loading.hidden = true;
    content.hidden = false;
    if (!vehicles.length) {
      tabs.innerHTML = '<h2>Your vehicles</h2>';
      target.innerHTML = '<div class="vehicle-empty-state vehicle-empty-state--large"><h2>Add your first vehicle</h2><p>Register the basics to start recording battery health and service history.</p><a class="btn btn--primary" href="/member/submit-vehicle-data/">Add vehicle</a></div>';
      return;
    }
    if (!vehicles.some(function (vehicle) { return vehicle.id === activeVehicleId; })) activeVehicleId = vehicles[0].id;
    tabs.innerHTML = vehicles.map(function (vehicle) {
      var selected = vehicle.id === activeVehicleId;
      return '<button id="vehicle-tab-' + escapeHtml(vehicle.id) + '" class="vehicle-tab" type="button" role="tab" aria-selected="' + selected + '" tabindex="' + (selected ? '0' : '-1') + '" data-vehicle-tab="' + escapeHtml(vehicle.id) + '">' + escapeHtml(vehicleName(vehicle)) + '</button>';
    }).join('');
    renderActiveVehicle(vehicles.find(function (vehicle) { return vehicle.id === activeVehicleId; }));
  }

  function openPanel(name) {
    workspace.querySelectorAll('[data-soh-panel], [data-event-panel]').forEach(function (panel) { panel.hidden = true; });
    var panel = workspace.querySelector(name === 'soh' ? '[data-soh-panel]' : '[data-event-panel]');
    if (panel) {
      panel.hidden = false;
      var first = panel.querySelector('input:not([type="hidden"]), select, textarea');
      if (first) first.focus();
    }
  }

  function editEvent(id) {
    var item = (memberData.serviceEvents || []).find(function (event) { return event.id === id; });
    if (!item) return;
    var form = workspace.querySelector('[data-service-event-form]');
    form.elements.id.value = item.id;
    form.elements.eventType.value = item.eventType;
    form.elements.occurredAt.value = item.occurredAt;
    form.elements.mileage.value = item.mileage == null ? '' : item.mileage;
    form.querySelectorAll('input[name="campaigns"]').forEach(function (input) {
      input.checked = (item.campaigns || []).indexOf(input.value) !== -1;
    });
    form.elements.finalFixAt.value = item.finalFixAt || '';
    form.elements.daysToFinalFix.value = item.daysToFinalFix == null ? '' : item.daysToFinalFix;
    form.elements.courtesyVehicleOffered.value = item.courtesyVehicleOffered || '';
    form.elements.courtesyVehicleProvided.value = item.courtesyVehicleProvided || '';
    form.elements.partsDelay.value = item.partsDelay || '';
    form.elements.warrantyCover.value = item.warrantyCover || '';
    form.elements.disputeStatus.value = item.disputeStatus || '';
    form.elements.title.value = item.title;
    form.elements.description.value = item.description || '';
    form.elements.status.value = item.status;
    form.querySelector('[data-event-form-title]').textContent = 'Edit service event or fault';
    openPanel('event');
  }

  function dateIsFuture(value) {
    return !!value && value > todayString();
  }

  function validateNotFutureDates(form) {
    var valid = true;
    form.querySelectorAll('input[type="date"][data-not-future]').forEach(function (input) {
      var error = input.parentNode.querySelector('[role="alert"]');
      var future = dateIsFuture(input.value);
      input.setAttribute('aria-invalid', future ? 'true' : 'false');
      if (error) error.hidden = !future;
      if (future && valid) {
        input.focus();
        valid = false;
      }
    });
    return valid;
  }

  workspace.addEventListener('click', function (event) {
    var tab = event.target.closest('[data-vehicle-tab]');
    var open = event.target.closest('[data-open-panel]');
    var close = event.target.closest('[data-close-panel]');
    var edit = event.target.closest('[data-edit-event]');
    if (tab) {
      activeVehicleId = tab.dataset.vehicleTab;
      render();
      Array.from(workspace.querySelectorAll('[data-vehicle-tab]')).find(function (item) {
        return item.dataset.vehicleTab === activeVehicleId;
      }).focus();
    } else if (open) {
      if (open.dataset.openPanel === 'event') {
        var form = workspace.querySelector('[data-service-event-form]');
        form.reset();
        form.elements.vehicleId.value = activeVehicleId;
        form.querySelector('[data-event-form-title]').textContent = 'Add service event or fault';
      }
      openPanel(open.dataset.openPanel);
    } else if (close) {
      var panel = workspace.querySelector(close.dataset.closePanel === 'soh' ? '[data-soh-panel]' : '[data-event-panel]');
      if (panel) panel.hidden = true;
    } else if (edit) {
      editEvent(edit.dataset.editEvent);
    }
  });

  workspace.addEventListener('keydown', function (event) {
    var tab = event.target.closest('[data-vehicle-tab]');
    if (!tab || (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight')) return;
    var tabs = Array.from(workspace.querySelectorAll('[data-vehicle-tab]'));
    var offset = event.key === 'ArrowRight' ? 1 : -1;
    var next = tabs[(tabs.indexOf(tab) + offset + tabs.length) % tabs.length];
    event.preventDefault();
    next.click();
  });

  workspace.addEventListener('submit', function (event) {
    var form = event.target.closest('[data-service-event-form]');
    if (!form) return;
    event.preventDefault();
    var status = form.querySelector('[data-service-event-status]');
    var button = form.querySelector('button[type="submit"]');
    var payload = {};
    new FormData(form).forEach(function (value, key) {
      if (key === 'campaigns') {
        if (!payload[key]) payload[key] = [];
        payload[key].push(value);
      } else {
        payload[key] = value;
      }
    });
    if (!validateNotFutureDates(form)) {
      if (status) status.textContent = 'Check the highlighted date before saving.';
      return;
    }
    button.disabled = true;
    status.textContent = 'Saving record...';
    Promise.resolve(window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : '').then(function (token) {
      return fetch('/api/upsert-service-event', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },
        body: JSON.stringify(payload),
      });
    }).then(function (response) {
      return response.json().catch(function () { return {}; }).then(function (data) {
        if (!response.ok || !data.ok) throw new Error(data.error || 'Could not save record');
      });
    }).then(function () {
      status.textContent = 'Record saved. Refreshing history...';
      return window.ipaceRefreshMemberData();
    }).catch(function (error) {
      status.textContent = error.message || 'Could not save the record.';
    }).finally(function () {
      button.disabled = false;
    });
  });

  document.addEventListener('member:data', function (event) {
    if (!event.detail || !event.detail.container.contains(workspace)) return;
    memberData = event.detail.data;
    render();
  });
})();
