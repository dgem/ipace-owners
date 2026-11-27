(function () {
  'use strict';
  function esc(value) { return String(value || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;'); }
  document.querySelectorAll('[data-public-survey-results]').forEach(function (root) {
    var id = root.getAttribute('data-public-survey-results');
    window.fetch('/api/public-survey-results?id=' + encodeURIComponent(id)).then(function (response) {
      if (!response.ok) throw new Error('Results are not available.');
      return response.json();
    }).then(function (data) {
      var total = Math.max(data.total || 0, 1);
      var html = '<div class="survey-results__header"><h2>' + esc(data.question) + '</h2><span>' + esc(data.total) + ' responses</span></div>';
      (data.options || []).forEach(function (option) {
        var count = (data.counts || {})[option.id] || 0;
        var preferred = (data.preferredCounts || {})[option.id] || 0;
        html += '<div class="survey-result survey-result--summary"><div><strong class="survey-result__name">' + esc(option.name) + '</strong><p class="survey-result__description">' + esc(option.description) + '</p><div class="survey-result__bar" aria-hidden="true"><span style="width: ' + (count / total * 100).toFixed(1) + '%"></span></div></div><div class="survey-result__count"><span class="survey-result__votes">' + esc(count) + ' selected</span><span class="survey-result__preferred">' + esc(preferred) + ' preferred</span></div></div>';
      });
      root.innerHTML = html;
    }).catch(function (error) { root.innerHTML = '<p class="form-hint">' + esc(error.message) + '</p>'; });
  });
}());
