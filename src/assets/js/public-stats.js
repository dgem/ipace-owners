(function () {
  'use strict';

  var roots = document.querySelectorAll('[data-public-stats]');
  if (!roots.length) return;

  function escapeHtml(value) {
    return String(value == null ? '' : value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function displayValue(value, format) {
    if (value == null) return 'Not collected';
    if (format === 'percent') return Number(value).toFixed(1) + '%';
    if (format === 'change') {
      var number = Number(value);
      return (number > 0 ? '+' : '') + number.toFixed(1) + ' pts';
    }
    return Number(value).toLocaleString('en-GB');
  }

  function renderDistribution(container, buckets) {
    if (!container) return;
    if (!buckets || !buckets.some(function (bucket) { return bucket.count > 0; })) {
      container.innerHTML = '<p class="text-muted">No data collected yet.</p>';
      return;
    }
    var max = Math.max.apply(null, buckets.map(function (bucket) { return bucket.count; }));
    container.innerHTML = buckets.map(function (bucket) {
      var width = max > 0 ? (bucket.count / max) * 100 : 0;
      return '<div class="bar-chart__row">' +
        '<span class="bar-chart__label">' + escapeHtml(bucket.label) + '</span>' +
        '<div class="bar-chart__track"><div class="bar-chart__fill" style="width:' + width.toFixed(1) + '%"></div></div>' +
        '<span class="bar-chart__value">' + Number(bucket.count).toLocaleString('en-GB') + '</span>' +
        '</div>';
    }).join('');
  }

  function render(root, data) {
    root.querySelectorAll('[data-public-stat]').forEach(function (element) {
      var key = element.getAttribute('data-public-stat');
      var value = data[key];
      element.textContent = displayValue(value, element.getAttribute('data-public-stat-format'));
      if (element.classList.contains('launch-member-count__value') && Number.isFinite(Number(value))) {
        var count = Math.max(0, Math.round(Number(value)));
        var countSize = count >= 100000 ? 'large' : count >= 10000 ? 'five' : count >= 1000 ? 'four' : 'standard';
        element.setAttribute('data-count-size', countSize);
      }
    });
    root.querySelectorAll('[data-public-generated-at]').forEach(function (element) {
      var generated = new Date(data.generatedAt);
      element.textContent = Number.isNaN(generated.getTime())
        ? 'Update time unavailable'
        : 'Updated ' + generated.toLocaleString('en-GB');
    });
    renderDistribution(root.querySelector('[data-public-distribution="soh"]'), data.sohDistribution);
    renderDistribution(root.querySelector('[data-public-distribution="model-year"]'), data.modelYearDistribution);
  }

  fetch('/api/public-stats?v=3')
    .then(function (response) {
      if (!response.ok) throw new Error('Public statistics returned ' + response.status);
      return response.json();
    })
    .then(function (data) {
      roots.forEach(function (root) { render(root, data); });
    })
    .catch(function (error) {
      console.warn('[public-stats] Could not load aggregate data.', error);
      roots.forEach(function (root) {
        root.querySelectorAll('[data-public-stat]').forEach(function (element) {
          element.textContent = 'Unavailable';
        });
        root.querySelectorAll('[data-public-distribution]').forEach(function (element) {
          element.innerHTML = '<p class="text-muted">Statistics are temporarily unavailable.</p>';
        });
        var status = root.querySelector('[data-public-stats-status]');
        if (status) status.textContent = 'Live statistics are temporarily unavailable.';
      });
    });
})();
