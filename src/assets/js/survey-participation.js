(function () {
  'use strict';
  var root = document.querySelector('[data-survey-participation]');
  if (!root) return;
  fetch('/api/survey-participation').then(function (response) {
    if (!response.ok) throw new Error('Count unavailable');
    return response.json();
  }).then(function (data) {
    if (!Number.isSafeInteger(data.responses) || data.responses < 0) return;
    root.querySelector('.launch-member-count__value').textContent = data.responses.toLocaleString('en-GB');
    root.querySelector('.launch-member-count__date').textContent = 'Latest response total';
  }).catch(function () {
    // Keep the explicitly dated, verified publication snapshot.
  });
})();
