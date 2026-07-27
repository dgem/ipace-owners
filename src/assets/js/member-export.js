/**
 * Authenticated member data downloads.
 */
(function () {
  'use strict';

  function filenameFromResponse(response, format) {
    var disposition = response.headers.get('Content-Disposition') || '';
    var match = disposition.match(/filename="([^"]+)"/i);
    if (match && match[1]) return match[1];
    return 'ipace-owner-data.' + (format === 'csv' ? 'zip' : 'xlsx');
  }

  function errorMessage(response) {
    return response.json().catch(function () { return {}; }).then(function (body) {
      return body.error || 'The export could not be prepared. Please try again.';
    });
  }

  function download(format, button, status) {
    var tokenPromise = window.ipaceGetIdentityToken ?
      window.ipaceGetIdentityToken() :
      Promise.resolve('');

    button.disabled = true;
    status.textContent = format === 'csv' ? 'Preparing CSV files…' : 'Preparing Excel report…';

    tokenPromise.then(function (token) {
      if (!token) throw new Error('Sign in again to export your data.');
      return fetch('/api/member-export?format=' + encodeURIComponent(format), {
        method: 'GET',
        headers: { Authorization: 'Bearer ' + token },
        credentials: 'same-origin',
      });
    }).then(function (response) {
      if (!response.ok) {
        return errorMessage(response).then(function (message) {
          throw new Error(message);
        });
      }
      return response.blob().then(function (blob) {
        return { blob: blob, filename: filenameFromResponse(response, format) };
      });
    }).then(function (file) {
      var url = URL.createObjectURL(file.blob);
      var link = document.createElement('a');
      link.href = url;
      link.download = file.filename;
      link.hidden = true;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(function () { URL.revokeObjectURL(url); }, 0);
      status.textContent = 'Your download is ready.';
    }).catch(function (error) {
      status.textContent = error.message || 'The export could not be prepared. Please try again.';
    }).finally(function () {
      button.disabled = false;
    });
  }

  function init() {
    var status = document.querySelector('[data-member-export-status]');
    if (!status) return;
    document.querySelectorAll('[data-member-export]').forEach(function (button) {
      button.addEventListener('click', function () {
        download(button.getAttribute('data-member-export'), button, status);
      });
    });
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
