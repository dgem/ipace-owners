(function () {
  'use strict';
  var button = document.querySelector('[data-service-export]');
  var status = document.querySelector('[data-service-export-status]');
  if (!button || !status) return;
  button.addEventListener('click', function () {
    if (button.disabled) return;
    button.disabled = true;
    status.textContent = 'Preparing service CSV…';
    var controller = new AbortController();
    var timer;
    var timeout = new Promise(function (_, reject) {
      timer = window.setTimeout(function () {
        controller.abort();
        reject(new Error('The export took too long. Please try again.'));
      }, 60000);
    });
    var download = Promise.resolve().then(function () {
      return window.ipaceGetIdentityToken ? window.ipaceGetIdentityToken() : '';
    }).then(function (token) {
      if (controller.signal.aborted) throw new Error('The export took too long. Please try again.');
      if (!token) throw new Error('Sign in again to export service data.');
      var headers = { Authorization: 'Bearer ' + token };
      return fetch('/api/admin/service-export', {
        headers: window.ipaceAuthHeaders ? window.ipaceAuthHeaders(headers) : headers,
        signal: controller.signal
      });
    }).then(function (response) {
      if (!response.ok) throw new Error('Could not export service data. Check your admin access and try again.');
      return response.blob();
    });
    Promise.race([download, timeout]).then(function (blob) {
      var url = URL.createObjectURL(blob);
      var link = document.createElement('a');
      link.href = url;
      link.download = 'ipace-service-data-redacted.csv';
      link.hidden = true;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(function () { URL.revokeObjectURL(url); }, 1000);
      status.textContent = 'Redacted service CSV downloaded.';
    }).catch(function (error) {
      status.textContent = controller.signal.aborted ? 'The export took too long. Please try again.' : error.message;
    }).finally(function () {
      window.clearTimeout(timer);
      button.disabled = false;
    });
  });
})();
