/**
 * member-auth.js — Server-side authentication verification.
 *
 * Replaces client-side auth gating with server-verified data loading.
 *
 * Usage in templates:
 *
 *   <!-- Member-gated page -->
 *   <div data-auth-container>
 *     <div class="auth-gate" data-auth-login-gate>
 *       <!-- Login/signup prompt — shown by default when not authenticated -->
 *     </div>
 *     <div class="auth-content" data-auth-content hidden>
 *       <!-- Content only revealed after server confirms auth -->
 *       <div data-member-data hidden></div>
 *     </div>
 *   </div>
 *
 *   <!-- Admin-gated page -->
 *   <div data-admin-container>
 *     <div class="auth-gate" data-auth-login-gate>
 *       <!-- Login prompt -->
 *     </div>
 *     <div class="auth-content" data-admin-content hidden>
 *       <div data-admin-data hidden></div>
 *     </div>
 *   </div>
 *
 * This script:
 * 1. Finds [data-auth-container] or [data-admin-container].
 * 2. Fetches the appropriate Firebase/GCP API (member-data or admin-data).
 * 3. On 200: hides the gate, shows content, and populates data.
 * 4. On 401/403: keeps the gate visible (login required).
 */

(function () {
  'use strict';

  // ── Member auth ──────────────────────────────────────────────────────────────
  var authRunId = 0;
  var adminRunId = 0;

  function showMemberGate(container) {
    var gate = container.querySelector('[data-auth-login-gate]');
    var content = container.querySelector('[data-auth-content]');
    var pending = container.querySelector('[data-auth-pending]');
    if (pending) pending.hidden = true;
    if (gate) gate.hidden = false;
    if (content) content.hidden = true;
  }

  function hideMemberGate(container) {
    var gate = container.querySelector('[data-auth-login-gate]');
    var content = container.querySelector('[data-auth-content]');
    var pending = container.querySelector('[data-auth-pending]');
    if (pending) pending.hidden = true;
    if (gate) gate.hidden = true;
    if (content) content.hidden = false;
  }

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
    var date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleDateString('en-GB');
  }

  function formatRelationship(value) {
    var labels = {
      'current-owner-one': 'Current owner of one I-PACE',
      'current-owner-multiple': 'Current owner of more than one I-PACE',
      'former-owner': 'Former owner',
      'prospective-buyer': 'Prospective buyer',
      'helping-owner': 'Helping an owner',
      'trade-specialist': 'Trade / specialist',
      other: 'Other',
      'current-owner': 'Current owner',
      prospective: 'Prospective buyer',
      trade: 'Trade / specialist',
    };
    return labels[value] || String(value || '').replace(/-/g, ' ');
  }

  function getIdentityToken() {
    if (window.ipaceGetIdentityToken) return window.ipaceGetIdentityToken();
    return Promise.resolve('');
  }

  function fetchWithIdentity(url, options) {
    return getIdentityToken().then(function (token) {
      options = options || {};
      options.headers = options.headers || {};
      if (token) {
        options.headers.Authorization = 'Bearer ' + token;
      }
      return fetch(url, options);
    });
  }

  function todayString() {
    var date = new Date();
    return date.getFullYear() + '-' + String(date.getMonth() + 1).padStart(2, '0') + '-' + String(date.getDate()).padStart(2, '0');
  }

  function validateNotFutureDates(form) {
    var valid = true;
    form.querySelectorAll('input[type="date"][data-not-future]').forEach(function (input) {
      var error = input.parentNode.querySelector('[role="alert"]');
      var future = !!input.value && input.value > todayString();
      input.setAttribute('aria-invalid', future ? 'true' : 'false');
      if (error) error.hidden = !future;
      if (future && valid) {
        input.focus();
        valid = false;
      }
    });
    return valid;
  }

  function populateVehicleRecords(container, records, readings) {
    var vehicleList = container.querySelector('[data-vehicle-list]');
    if (!vehicleList) return;

    if (!records || records.length === 0) {
      vehicleList.innerHTML =
        '<p class="empty-state" style="color:var(--color-text-muted);">No vehicle data submitted yet. ' +
        '<a href="/submit-vehicle-data/">Add your first vehicle</a>.</p>';
      return;
    }

    var html = '<div class="account-vehicle-grid">';
    records.forEach(function (rec) {
      var veh = rec.vehicle || {};
      var bat = rec.battery || {};
      var vehicleReadings = (readings || []).filter(function (reading) {
        return reading.vehicleId === rec.id;
      });
      var latestReading = vehicleReadings.length ? vehicleReadings[vehicleReadings.length - 1].battery || {} : null;
      var latestSOH = latestReading && latestReading.stateOfHealth != null ? latestReading.stateOfHealth : bat.stateOfHealth;
      var latestDate = latestReading && latestReading.measuredAt ? latestReading.measuredAt : bat.measuredAt;

      html += '<article class="account-vehicle-card">';
      html += '<div class="account-vehicle-card__main"><h3>';
      if (veh.registration) html += escapeHtml(veh.registration);
      else html += 'Vehicle ' + escapeHtml(rec.id.slice(-6));
      html += '</h3>';

      var details = [];
      if (veh.country) details.push('Country: ' + escapeHtml(veh.country));
      if (veh.modelYear) details.push('Model year: ' + escapeHtml(veh.modelYear));
      if (veh.mileage != null) details.push('Mileage: ' + Number(veh.mileage).toLocaleString());
      if (veh.ownedSince) details.push('Owned since: ' + escapeHtml(veh.ownedSince));

      if (details.length > 0) {
        html += '<p class="account-vehicle-card__meta">' + details.join(' &middot; ') + '</p>';
      } else {
        html += '<p class="account-vehicle-card__meta">Vehicle details are ready to expand.</p>';
      }

      html += '<div class="account-vehicle-card__footer">';
      html += '<span>Submitted ' + escapeHtml(formatDate(rec.createdAt)) + '</span>';
      if (rec.updatedAt !== rec.createdAt) {
        html += '<span>Updated ' + escapeHtml(formatDate(rec.updatedAt)) + '</span>';
      }
      html += '</div>';
      html += '</div>';

      html += '<div class="account-vehicle-card__stats">';
      if (latestSOH != null) {
        html += '<p class="account-vehicle-card__stat"><strong>' + escapeHtml(latestSOH) + '%</strong><span>Latest SoH</span></p>';
        if (latestDate) html += '<p class="account-vehicle-card__hint">Measured ' + escapeHtml(formatDate(latestDate)) + '</p>';
      } else {
        html += '<p class="account-vehicle-card__stat"><strong>—</strong><span>No SoH yet</span></p>';
      }
      html += '<p class="account-vehicle-card__hint">' + vehicleReadings.length + ' SoH ' + (vehicleReadings.length === 1 ? 'reading' : 'readings') + '</p>';
      html += '<a class="btn btn--secondary btn--sm" href="/member/dashboard/">Manage history</a>';
      html += '</div>';
      html += '</article>';
    });
    html += '</div>';
    vehicleList.innerHTML = html;
   }

  function populateJoinInfo(container, records) {
    var joinEl = container.querySelector('[data-join-info]');
    if (!joinEl || !records || records.length === 0) return;

    var rec = records[0];
    var contact = rec.contact || {};
    var membership = rec.membership || {};

    var html = '<div class="join-info" style="font-size:var(--text-sm);">';
    if (contact.name) html += '<p><strong>Name:</strong> ' + escapeHtml(contact.name) + '</p>';
    if (contact.country) html += '<p><strong>Country:</strong> ' + escapeHtml(contact.country) + '</p>';
    if (membership.relationship) html += '<p><strong>Relationship:</strong> ' + escapeHtml(formatRelationship(membership.relationship)) + '</p>';
    if (rec.createdAt) html += '<p><strong>Joined:</strong> ' + escapeHtml(formatDate(rec.createdAt)) + '</p>';
    html += '</div>';
    joinEl.innerHTML = html;
   }

  function preferenceRow(label, enabled, enabledLabel, disabledLabel) {
    var statusClass = enabled ? 'preference-list__status preference-list__status--on' : 'preference-list__status';
    var statusText = enabled ? enabledLabel : disabledLabel;
    return '<div class="preference-list__row">' +
      '<dt>' + escapeHtml(label) + '</dt>' +
      '<dd><span class="' + statusClass + '">' + escapeHtml(statusText) + '</span></dd>' +
      '</div>';
  }

  function populatePreferences(container, records) {
    var preferencesEl = container.querySelector('[data-preferences-container]');
    if (!preferencesEl) return;

    if (!records || records.length === 0) {
      preferencesEl.innerHTML =
        '<p class="form-hint">No membership preferences have been recorded yet. Join the group to set contact and data-use preferences.</p>';
      return;
    }

    var rec = records[0];
    var consents = rec.consents || {};
    var membership = rec.membership || {};
    var skills = membership.skills || [];

    var html = '<dl class="preference-list">';
    html += preferenceRow('Group contact', consents.contact, 'Enabled', 'Not enabled');
    html += preferenceRow('Anonymised aggregate analysis', consents.anonymisedAnalysis, 'Allowed', 'Not allowed');
    html += preferenceRow('Participation acknowledgement', consents.notLegalClaim, 'Confirmed', 'Not recorded');
    if (membership.relationship) {
      html += '<div class="preference-list__row"><dt>Relationship</dt><dd>' + escapeHtml(formatRelationship(membership.relationship)) + '</dd></div>';
    }
    if (skills.length > 0) {
      html += '<div class="preference-list__row"><dt>Volunteering interests</dt><dd>' + skills.map(escapeHtml).join(', ') + '</dd></div>';
    }
    if (rec.createdAt) {
      html += '<div class="preference-list__row"><dt>Recorded</dt><dd>' + escapeHtml(formatDate(rec.createdAt)) + '</dd></div>';
    }
    html += '</dl>';
    html += '<p class="form-hint">Preference editing will be added with an audited account update flow. For now, contact the group if these details need correcting.</p>';
    preferencesEl.innerHTML = html;
   }

  async function verifyMemberAuth() {
    var runId = ++authRunId;
    var containers = document.querySelectorAll('[data-auth-container]');
    containers.forEach(async function (container) {
      try {
        var res = await fetchWithIdentity('/api/member-data');
        if (runId !== authRunId) return;
        if (res.status === 401 || res.status === 403) {
          showMemberGate(container);
          return;
         }

        if (!res.ok) {
          console.warn('[member-auth] Unexpected status:', res.status);
          showMemberGate(container);
          return;
         }

        var data = await res.json();
        hideMemberGate(container);

        // Populate vehicle records
        var vehicleContainer = container.querySelector('[data-vehicle-container]');
        if (vehicleContainer && data.vehicleRecords) {
          populateVehicleRecords(vehicleContainer, data.vehicleRecords, data.batteryReadings || []);
        }

        var joinContainer = container.querySelector('[data-join-container]');
        if (joinContainer && data.joinRecords) {
          populateJoinInfo(joinContainer, data.joinRecords);
        }

        populatePreferences(container, data.joinRecords || []);

        // Expose raw data for other scripts to consume
        container.dataset.memberData = JSON.stringify(data);
        document.dispatchEvent(new CustomEvent('member:data', {
          detail: { container: container, data: data },
        }));
       } catch (err) {
      console.warn('[member-auth] Failed to verify auth:', err);
      showMemberGate(container);
     }
    });
   }

  window.ipaceRefreshMemberData = verifyMemberAuth;

   // ── Admin auth ───────────────────────────────────────────────────────────────

  function showAdminGate(container) {
    var gate = container.querySelector('[data-auth-login-gate]');
    var adminOnlyGate = container.querySelector('[data-admin-only-gate]');
    var content = container.querySelector('[data-admin-content]');
    var pending = container.querySelector('[data-auth-pending]');

    if (pending) pending.hidden = true;
    if (gate) gate.hidden = false;
    if (adminOnlyGate) adminOnlyGate.hidden = true;
    if (content) content.hidden = true;
   }

  function hideAdminGate(container) {
    var gate = container.querySelector('[data-auth-login-gate]');
    var adminOnlyGate = container.querySelector('[data-admin-only-gate]');
    var content = container.querySelector('[data-admin-content]');
    var pending = container.querySelector('[data-auth-pending]');

    if (pending) pending.hidden = true;
    if (gate) gate.hidden = true;
    if (adminOnlyGate) adminOnlyGate.hidden = true;
    if (content) content.hidden = false;
   }

  function populateAdminStats(container, data) {
    var statsEl = container.querySelector('[data-admin-stats]');
    if (!statsEl) return;

    var totalJoin = (data.joinRecords || []).length;
    var totalVehicle = (data.vehicleRecords || []).length;

    statsEl.innerHTML =
      '<div class="stat-card" style="display:inline-block; padding:var(--space-4); margin-right:var(--space-4); text-align:center;">' +
        '<strong style="font-size:var(--text-2xl);">' + totalJoin + '</strong>' +
        '<p style="margin:0; font-size:var(--text-sm); color:var(--color-text-muted);">Join submissions</p>' +
       '</div>' +
      '<div class="stat-card" style="display:inline-block; padding:var(--space-4); text-align:center;">' +
        '<strong style="font-size:var(--text-2xl);">' + totalVehicle + '</strong>' +
        '<p style="margin:0; font-size:var(--text-sm); color:var(--color-text-muted);">Vehicle records</p>' +
       '</div>';
   }

  function populateAdminJoinTable(container, records) {
    var tableEl = container.querySelector('[data-admin-join-table]');
    if (!tableEl) return;

    if (!records || records.length === 0) {
      tableEl.innerHTML = '<p class="empty-state" style="color:var(--color-text-muted);">No join submissions yet.</p>';
      return;
     }

    var html = '<table style="width:100%; border-collapse:collapse; font-size:var(--text-sm);">';
    html += '<thead><tr style="border-bottom:2px solid var(--color-border); text-align:left;">';
    html += '<th style="padding:var(--space-2);">Name</th>';
    html += '<th style="padding:var(--space-2);">Email</th>';
    html += '<th style="padding:var(--space-2);">Country</th>';
    html += '<th style="padding:var(--space-2);">Relationship</th>';
    html += '<th style="padding:var(--space-2);">Status</th>';
    html += '<th style="padding:var(--space-2);">Submitted</th>';
    html += '</tr></thead><tbody>';

    records.forEach(function (rec) {
      var contact = rec.contact || {};
      var membership = rec.membership || {};
      var review = rec.review || {};
      html += '<tr style="border-bottom:1px solid var(--color-border);">';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(contact.name || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(contact.email || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(contact.country || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(membership.relationship ? formatRelationship(membership.relationship) : '—') + '</td>';
      html += '<td style="padding:var(--space-2);"><span class="badge">' + escapeHtml(review.status || 'new') + '</span></td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(formatDate(rec.createdAt)) + '</td>';
      html += '</tr>';
     });

    html += '</tbody></table>';
    tableEl.innerHTML = html;
   }

  function populateAdminVehicleTable(container, records) {
    var tableEl = container.querySelector('[data-admin-vehicle-table]');
    if (!tableEl) return;

    if (!records || records.length === 0) {
      tableEl.innerHTML = '<p class="empty-state" style="color:var(--color-text-muted);">No vehicle records yet.</p>';
      return;
     }

    var html = '<table style="width:100%; border-collapse:collapse; font-size:var(--text-sm);">';
    html += '<thead><tr style="border-bottom:2px solid var(--color-border); text-align:left;">';
    html += '<th style="padding:var(--space-2);">Registration</th>';
    html += '<th style="padding:var(--space-2);">Country</th>';
    html += '<th style="padding:var(--space-2);">Model Year</th>';
    html += '<th style="padding:var(--space-2);">Mileage</th>';
    html += '<th style="padding:var(--space-2);">SoH</th>';
    html += '<th style="padding:var(--space-2);">Status</th>';
    html += '<th style="padding:var(--space-2);">Submitted</th>';
    html += '</tr></thead><tbody>';

    records.forEach(function (rec) {
      var veh = rec.vehicle || {};
      var bat = rec.battery || {};
      var review = rec.review || {};
      html += '<tr style="border-bottom:1px solid var(--color-border);">';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(veh.registration || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(veh.country || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(veh.modelYear || '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + (veh.mileage != null ? Number(veh.mileage).toLocaleString() : '—') + '</td>';
      html += '<td style="padding:var(--space-2);">' + (bat.stateOfHealth != null ? bat.stateOfHealth + '%' : '—') + '</td>';
      html += '<td style="padding:var(--space-2);"><span class="badge">' + escapeHtml(review.status || 'new') + '</span></td>';
      html += '<td style="padding:var(--space-2);">' + escapeHtml(formatDate(rec.createdAt)) + '</td>';
      html += '</tr>';
     });

    html += '</tbody></table>';
    tableEl.innerHTML = html;
   }

  async function verifyAdminAuth() {
    var runId = ++adminRunId;
    var containers = document.querySelectorAll('[data-admin-container]');
    containers.forEach(async function (container) {
      try {
        var res = await fetchWithIdentity('/api/admin-data');
        if (runId !== adminRunId) return;

         // 401 = not logged in → show login gate
        if (res.status === 401) {
          showAdminGate(container);
          return;
         }

         // 403 = logged in but not admin → show admin-only gate
        if (res.status === 403) {
          var gate = container.querySelector('[data-auth-login-gate]');
          var adminOnlyGate = container.querySelector('[data-admin-only-gate]');
          var content = container.querySelector('[data-admin-content]');
          var pending = container.querySelector('[data-auth-pending]');
          if (pending) pending.hidden = true;
          if (gate) gate.hidden = true;
          if (adminOnlyGate) adminOnlyGate.hidden = false;
          if (content) content.hidden = true;
          return;
         }

        if (!res.ok) {
          console.warn('[member-auth] Unexpected status for admin:', res.status);
          showAdminGate(container);
          return;
         }

        var data = await res.json();
        hideAdminGate(container);

         // Populate stats, tables, etc.
        var statsContainer = container.querySelector('[data-stats-container]');
        if (statsContainer) {
          populateAdminStats(statsContainer, data);
         }

        var joinTableContainer = container.querySelector('[data-join-table-container]');
        if (joinTableContainer && data.joinRecords) {
          populateAdminJoinTable(joinTableContainer, data.joinRecords);
         }

        var vehicleTableContainer = container.querySelector('[data-vehicle-table-container]');
        if (vehicleTableContainer && data.vehicleRecords) {
          populateAdminVehicleTable(vehicleTableContainer, data.vehicleRecords);
         }

         // Expose raw data for other scripts
        container.dataset.adminData = JSON.stringify(data);
       } catch (err) {
      console.warn('[member-auth] Failed to verify admin auth:', err);
      showAdminGate(container);
     }
    });
   }

   // ── Init on DOM ready ────────────────────────────────────────────────────────

  function init() {
    if (document.querySelectorAll('[data-auth-container]').length > 0) {
      verifyMemberAuth();
     }
    if (document.querySelectorAll('[data-admin-container]').length > 0) {
      verifyAdminAuth();
     }
   }

  function initSoon() {
    window.setTimeout(init, 0);
  }

  document.addEventListener('submit', function (event) {
    var form = event.target.closest('[data-soh-update-form]');
    if (!form) return;
    event.preventDefault();
    var status = form.querySelector('[data-soh-update-status]');
    var button = form.querySelector('button[type="submit"]');
    var payload = {};
    new FormData(form).forEach(function (value, key) { payload[key] = value; });
    if (!validateNotFutureDates(form)) {
      if (status) status.textContent = 'Check the highlighted date before saving.';
      return;
    }
    if (button) button.disabled = true;
    if (status) status.textContent = 'Saving reading...';

    fetchWithIdentity('/api/submit-soh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    }).then(function (response) {
      return response.json().catch(function () { return {}; }).then(function (data) {
        if (!response.ok || !data.ok) throw new Error(data.error || 'Could not save reading');
        return data;
      });
    }).then(function () {
      if (status) status.textContent = 'Reading saved. Refreshing history...';
      verifyMemberAuth();
    }).catch(function (error) {
      console.warn('[member-auth] SoH update failed:', error);
      if (status) status.textContent = error.message || 'Could not save the reading.';
    }).finally(function () {
      if (button) button.disabled = false;
    });
  });

  function initWhenIdentityReady() {
    if (window.ipaceIdentityReady) {
      initSoon();
      return;
    }

    // If the auth adapter never emits init, do not leave gated pages stuck
    // in their pending state. The server check will show the login gate if no
    // token is available.
    window.setTimeout(function () {
      if (!window.ipaceIdentityReady) init();
    }, 1500);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initWhenIdentityReady);
   } else {
    initWhenIdentityReady();
   }

  document.addEventListener('identity:ready', initSoon);
  document.addEventListener('identity:login', initSoon);
  document.addEventListener('identity:logout', initSoon);

})();
