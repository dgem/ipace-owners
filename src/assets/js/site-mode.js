/**
 * site-mode.js — Select the public launch or complete site presentation.
 *
 * This is a discoverability flag, not an authorization boundary. It runs
 * synchronously in <head> so full-only content does not flash in launch mode.
 */

(function () {
  'use strict';

  var root = document.documentElement;
  var storageKey = 'ipace_site_mode';
  var allowedModes = ['launch', 'full'];
  var defaultMode = allowedModes.indexOf(root.getAttribute('data-site-mode')) !== -1
    ? root.getAttribute('data-site-mode')
    : 'launch';
  var requestedMode = '';
  var storedMode = '';

  try {
    requestedMode = new URLSearchParams(window.location.search).get('site-mode') || '';
  } catch (error) {
    requestedMode = '';
  }

  if (allowedModes.indexOf(requestedMode) !== -1) {
    try {
      window.sessionStorage.setItem(storageKey, requestedMode);
    } catch (error) {
      // Storage can be unavailable; the requested page still uses this mode.
    }
    root.setAttribute('data-site-mode', requestedMode);
    return;
  }

  try {
    storedMode = window.sessionStorage.getItem(storageKey) || '';
  } catch (error) {
    storedMode = '';
  }

  root.setAttribute(
    'data-site-mode',
    allowedModes.indexOf(storedMode) !== -1 ? storedMode : defaultMode
  );
})();
