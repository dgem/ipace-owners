/**
 * admin-stats.js — Client-side admin statistics dashboard.
 *
 * Fetches /api/admin/stats, renders member/vehicle/service stats with
 * daily line charts (Canvas API), data tables and stat cards.
 *
 * Usage: Load on admin pages via defer script tag.
 */

(function () {
  'use strict';

  var NS = '[admin-stats] ';
  var API_ENDPOINT = '/api/admin/stats';
  var TABLE_ROWS_LIMIT = 200;

  // ── Helpers ───────────────────────────────────────────────────────────────

  function escapeHtml(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  // ── Fetch & Render ────────────────────────────────────────────────────────

  async function fetchAdminStats(container) {
    var descriptionEl = container.querySelector('[data-admin-stats-description]');
    if (descriptionEl) descriptionEl.textContent = 'Fetching statistics…';

    try {
      var token = window.ipaceGetIdentityToken ? await window.ipaceGetIdentityToken() : '';
      if (!token) throw new Error('Sign in as an administrator first.');
      var response = await window.fetch(API_ENDPOINT, {
        method: 'GET',
        headers: { 'Accept': 'application/json', 'Authorization': 'Bearer ' + token }
      });

      if (!response.ok) {
        throw new Error('HTTP ' + response.status + ': ' + response.statusText);
      }

      var data = await response.json();
      if (descriptionEl) descriptionEl.textContent = 'Loaded on ' + new Date().toLocaleDateString('en-GB') + ' at ' + new Date().toLocaleTimeString('en-GB');
      renderAllStats(container, data);
    } catch (err) {
      console.error(NS, err);
      if (descriptionEl) descriptionEl.textContent = 'Failed to load statistics: ' + escapeHtml(err && err.message ? err.message : 'Unknown error');
    }
  }

  // ── Stat Cards ───────────────────────────────────────────────────────────

  function renderStatCards(container, selector, stats, dataAttrs) {
    var grid = container.querySelector(selector);
    if (!grid) return;

    grid.innerHTML = '';
    for (var i = 0; i < dataAttrs.length; i++) {
      var attr = dataAttrs[i];
      var card = document.createElement('article');
      card.className = 'stat-card' + (attr.accent ? ' stat-card--accent' : '');
      card.innerHTML =
        '<h3 class="stat-card__label">' + escapeHtml(attr.label) + '</h3>' +
        '<p class="stat-card__value" data-stat="' + escapeHtml(attr.key) + '">—</p>';
      grid.appendChild(card);
    }

    var secondaryEl = grid.querySelectorAll('[data-stat]');
    for (var j = 0; j < secondaryEl.length; j++) {
      var key = secondaryEl[j].getAttribute('data-stat');
      if (key && stats[key] != null) {
        secondaryEl[j].textContent = String(stats[key]);
      }
    }
  }

  // ── Timeline Graphs (Canvas API) ─────────────────────────────────────────

  function renderTimeline(canvas, buckets, title) {
    if (!canvas || !buckets || buckets.length === 0) {
      if (canvas) canvas.parentNode.innerHTML =
        '<p>No timeline data available.</p>';
      return;
    }

    var width = parseInt(canvas.getAttribute('width'), 10) || canvas.clientWidth || 600;
    var height = parseInt(canvas.getAttribute('height'), 10) || 300;
    var ctx = canvas.getContext('2d');

    // Clear canvas
    ctx.clearRect(0, 0, width, height);

    var padding = { top: 20, right: 20, bottom: 40, left: 50 };
    var chartWidth = width - padding.left - padding.right;
    var chartHeight = height - padding.top - padding.bottom;

    // Find max count
    var maxCount = 0;
    for (var i = 0; i < buckets.length; i++) {
      if (buckets[i].count > maxCount) maxCount = buckets[i].count;
    }
    if (maxCount === 0) maxCount = 1;

    // Draw axes
    ctx.strokeStyle = '#9ca3af';
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(padding.left, padding.top);
    ctx.lineTo(padding.left, height - padding.bottom);
    ctx.lineTo(width - padding.right, height - padding.bottom);
    ctx.stroke();

    // Y-axis labels
    ctx.fillStyle = '#4b5563';
    ctx.font = '11px system-ui, -apple-system, sans-serif';
    ctx.textAlign = 'right';
    var ySteps = 5;
    for (var step = 0; step <= ySteps; step++) {
      var value = Math.round(maxCount / ySteps * step);
      var yPos = height - padding.bottom - (chartHeight / ySteps * step);
      ctx.fillText(String(value), padding.left - 5, yPos + 4);
      ctx.strokeStyle = '#e5e7eb';
      ctx.beginPath();
      ctx.moveTo(padding.left, yPos);
      ctx.lineTo(width - padding.right, yPos);
      ctx.stroke();
    }

    // Daily trend line and labels
    var pointGap = buckets.length > 1 ? chartWidth / (buckets.length - 1) : 0;
    ctx.textAlign = 'center';
    ctx.font = '10px system-ui, -apple-system, sans-serif';
    ctx.beginPath();
    for (var b = 0; b < buckets.length; b++) {
      var bucket = buckets[b];
      var x = padding.left + pointGap * b;
      var y = height - padding.bottom - (bucket.count / maxCount) * chartHeight;
      if (b === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    }
    ctx.strokeStyle = '#0f766e';
    ctx.lineWidth = 2;
    ctx.stroke();

    for (var pointIndex = 0; pointIndex < buckets.length; pointIndex++) {
      var pointBucket = buckets[pointIndex];
      var pointX = padding.left + pointGap * pointIndex;
      var pointY = height - padding.bottom - (pointBucket.count / maxCount) * chartHeight;
      ctx.fillStyle = '#0f766e';
      ctx.beginPath();
      ctx.arc(pointX, pointY, 3, 0, Math.PI * 2);
      ctx.fill();

      // X-axis label (rotate if needed)
      ctx.save();
      ctx.translate(pointX, height - padding.bottom + 12);
      if (buckets.length > 7) {
        ctx.rotate(-Math.PI / 4);
        ctx.textAlign = 'right';
      } else {
        ctx.textAlign = 'center';
      }
      ctx.fillStyle = '#4b5563';
      ctx.fillText(pointBucket.label, 0, 0);
      ctx.restore();
    }

    // Title
    ctx.fillStyle = '#12324a';
    ctx.font = 'bold 13px system-ui, -apple-system, sans-serif';
    ctx.textAlign = 'left';
    ctx.fillText(title, padding.left, 12);
  }

  // ── Data Tables ───────────────────────────────────────────────────────────

  // ── Service Event Aggregates Table ────────────────────────────────────────

  function renderServiceAggregates(container, aggregates) {
    var table = container.querySelector('[data-service-event-aggregates]');
    if (!table || !aggregates || aggregates.length === 0) {
      if (table) table.parentNode.innerHTML = '<p>No service event data available.</p>';
      return;
    }

    var tbody = table.querySelector('tbody');
    if (!tbody) return;
    tbody.innerHTML = '';

    for (var i = 0; i < Math.min(aggregates.length, TABLE_ROWS_LIMIT); i++) {
      var agg = aggregates[i];
      var tr = document.createElement('tr');
      var fields = [
        { key: 'category' },
        { key: 'eventCount', format: function(v) { return v == null ? '—' : String(v); } },
        { key: 'minDays', format: function(v) { return v == null ? '—' : String(v); } },
        { key: 'avgDays', format: function(v) { return v == null ? '—' : (v != null ? Math.round(v).toString() : '—'); } },
        { key: 'maxDays', format: function(v) { return v == null ? '—' : String(v); } }
      ];
      for (var f = 0; f < fields.length; f++) {
        var td = document.createElement('td');
        var val = agg[fields[f].key];
        td.textContent = fields[f].format ? fields[f].format(val) : (val == null ? '—' : String(val));
        tr.appendChild(td);
      }
      tbody.appendChild(tr);
    }
  }

  // ── Main Render Function ──────────────────────────────────────────────────

  function renderAllStats(container, data) {
    var memberStats = data.memberStats || {};
    var vehicleStats = data.vehicleStats || {};
    var serviceStats = data.serviceEventStats || {};

    // Render stat cards for members
    renderStatCards(container, '[data-member-stats]', memberStats, [
      { key: 'totalMembers', label: 'Total Members' },
      { key: 'verifiedCount', label: 'Verified Accounts', accent: true }
    ]);

    // Render country table
    if (memberStats.countryBreakup && memberStats.countryBreakup.length > 0) {
      var countryTable = container.querySelector('[data-country-table] tbody');
      if (countryTable) {
        countryTable.innerHTML = '';
        for (var c = 0; c < Math.min(memberStats.countryBreakup.length, TABLE_ROWS_LIMIT); c++) {
          var item = memberStats.countryBreakup[c];
          var countryRow = document.createElement('tr');
          countryRow.innerHTML =
            '<td>' + escapeHtml(item.country) + '</td>' +
            '<td>' + escapeHtml(item.joined) + '</td>' +
            '<td>' + escapeHtml(item.registered) + '</td>' +
            '<td>' + escapeHtml(item.verified) + '</td>';
          countryTable.appendChild(countryRow);
        }
      }
    }

    // Render join timeline
    var memberTimelineCanvas = container.querySelector('[data-member-timeline]');
    if (memberTimelineCanvas && memberStats.joinedTimeline && memberStats.joinedTimeline.length > 0) {
      renderTimeline(memberTimelineCanvas, memberStats.joinedTimeline, 'Members Joined by Day');
    }
    var verifiedTimelineCanvas = container.querySelector('[data-verified-timeline]');
    if (verifiedTimelineCanvas && memberStats.verifiedTimeline && memberStats.verifiedTimeline.length > 0) {
      renderTimeline(verifiedTimelineCanvas, memberStats.verifiedTimeline, 'Verified Accounts by Day');
    }

    // Render stat cards for vehicles
    renderStatCards(container, '[data-vehicle-stats]', vehicleStats, [
      { key: 'totalVehicles', label: 'Total Vehicles' },
      { key: 'vehiclesWithSoh', label: 'SoH Readings' }
    ]);

    // Render model year table
    if (vehicleStats.modelYearBreakup && vehicleStats.modelYearBreakup.length > 0) {
      var modelYearTable = container.querySelector('[data-model-year-table] tbody');
      if (modelYearTable) {
        modelYearTable.innerHTML = '';
        for (var m = 0; m < Math.min(vehicleStats.modelYearBreakup.length, TABLE_ROWS_LIMIT); m++) {
          var vItem = vehicleStats.modelYearBreakup[m];
          var modelYearRow = document.createElement('tr');
          modelYearRow.innerHTML =
            '<td>' + escapeHtml(vItem.modelYear) + '</td>' +
            '<td>' + escapeHtml(vItem.count) + '</td>';
          modelYearTable.appendChild(modelYearRow);
        }
      }
    }

    // Render service event aggregates
    if (serviceStats.categoryAggregates && serviceStats.categoryAggregates.length > 0) {
      renderServiceAggregates(container, serviceStats.categoryAggregates);
    } else {
      var serviceSection = container.querySelector('[data-service-event-aggregates]');
      if (serviceSection) {
        serviceSection.parentNode.innerHTML = '<p>No service event data available.</p>';
      }
    }
  }

  // ── Initialization ────────────────────────────────────────────────────────

  function init() {
    var containers = document.querySelectorAll('[data-admin-stats-section]');
    for (var i = 0; i < containers.length; i++) {
      (function (container) {
        var content = container.closest('[data-admin-container]').querySelector('[data-admin-content]');
        if (!content) return;
        var loadWhenVisible = function () {
          if (!content.hidden) {
            fetchAdminStats(container);
            observer.disconnect();
          }
        };
        var observer = new MutationObserver(loadWhenVisible);
        observer.observe(content, { attributes: true, attributeFilter: ['hidden'] });
        loadWhenVisible();
      }(containers[i]));
    }
  }

  // Run on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

  // Expose for debugging
  window.adminStats = { fetchAdminStats: fetchAdminStats, renderAllStats: renderAllStats };

})();
