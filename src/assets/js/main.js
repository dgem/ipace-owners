/**
 * main.js — Mobile menu toggle and general UI enhancements.
 */

(function () {
  'use strict';

  // ── Mobile menu toggle ──────────────────────────────────────────────────────
  const toggle = document.getElementById('mobile-menu-toggle');
  const mobileNav = document.getElementById('mobile-nav');

  if (toggle && mobileNav) {
    toggle.addEventListener('click', function () {
      const isOpen = mobileNav.classList.toggle('is-open');
      toggle.setAttribute('aria-expanded', String(isOpen));
      toggle.setAttribute(
        'aria-label',
        isOpen ? 'Close navigation menu' : 'Open navigation menu'
      );
    });

    // Close on Escape
    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && mobileNav.classList.contains('is-open')) {
        mobileNav.classList.remove('is-open');
        toggle.setAttribute('aria-expanded', 'false');
        toggle.setAttribute('aria-label', 'Open navigation menu');
        toggle.focus();
      }
    });

    // Close on outside click
    document.addEventListener('click', function (e) {
      if (
        mobileNav.classList.contains('is-open') &&
        !mobileNav.contains(e.target) &&
        !toggle.contains(e.target)
      ) {
        mobileNav.classList.remove('is-open');
        toggle.setAttribute('aria-expanded', 'false');
        toggle.setAttribute('aria-label', 'Open navigation menu');
      }
    });
  }

  // ── Mark current nav links ──────────────────────────────────────────────────
  const currentPath = window.location.pathname;
  document.querySelectorAll('[data-nav-link]').forEach(function (link) {
    const href = link.getAttribute('href');
    if (href && (href === currentPath || (href !== '/' && currentPath.startsWith(href)))) {
      link.setAttribute('aria-current', 'page');
    }
  });

  // ── Cookie notice ──────────────────────────────────────────────────────────
  const cookieNotice = document.querySelector('[data-cookie-notice]');
  const cookieAccept = document.querySelector('[data-cookie-accept]');
  const cookieStorageKey = 'ipace_cookie_notice_accepted';

  if (cookieNotice && cookieAccept) {
    try {
      if (window.localStorage.getItem(cookieStorageKey) !== 'true') {
        cookieNotice.hidden = false;
      }
    } catch (e) {
      cookieNotice.hidden = false;
    }

    cookieAccept.addEventListener('click', function () {
      try {
        window.localStorage.setItem(cookieStorageKey, 'true');
      } catch (e) {
        // Storage may be unavailable; still dismiss for the current page view.
      }
      cookieNotice.hidden = true;
    });
  }

})();
