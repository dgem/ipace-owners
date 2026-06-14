/**
 * identity.js — Netlify Identity UI integration.
 *
 * Initialises netlify-identity-widget and keeps the header UI in sync
 * with the current auth state. Also handles gated content areas.
 *
 * NOTE: Netlify Identity must be enabled in the Netlify UI after deployment
 * (Site Settings → Identity → Enable Identity). The widget is loaded from
 * the CDN in base.njk.
 */

(function () {
  'use strict';

  // netlify-identity-widget is loaded as a CDN script in base.njk.
  // It attaches itself to window.netlifyIdentity.
  var identity = window.netlifyIdentity;

  if (!identity) {
    console.warn('[identity.js] netlify-identity-widget not found. Identity features disabled.');
    return;
  }

  // ── Helper: get current user ────────────────────────────────────────────────
  function currentUser() {
    return identity.currentUser();
  }

  function isAdmin(user) {
    if (!user) return false;
    var roles = (user.app_metadata && user.app_metadata.roles) || [];
    return roles.indexOf('admin') !== -1;
  }

  // ── Header UI update ────────────────────────────────────────────────────────
  var loginBtn        = document.getElementById('identity-login-btn');
  var logoutBtn       = document.getElementById('identity-logout-btn');
  var userDisplay     = document.getElementById('identity-user-display');
  var mobileLoginBtn  = document.getElementById('identity-mobile-login-btn');
  var mobileLogoutBtn = document.getElementById('identity-mobile-logout-btn');

  function updateHeaderUI(user) {
    if (user) {
      // Logged in
      if (loginBtn)        loginBtn.style.display       = 'none';
      if (logoutBtn)       logoutBtn.style.display      = '';
      if (mobileLoginBtn)  mobileLoginBtn.style.display = 'none';
      if (mobileLogoutBtn) mobileLogoutBtn.style.display = '';
      if (userDisplay) {
        userDisplay.style.display   = '';
        userDisplay.textContent     = user.email || 'Member';
      }
    } else {
      // Logged out
      if (loginBtn)        loginBtn.style.display       = '';
      if (logoutBtn)       logoutBtn.style.display      = 'none';
      if (mobileLoginBtn)  mobileLoginBtn.style.display = '';
      if (mobileLogoutBtn) mobileLogoutBtn.style.display = 'none';
      if (userDisplay)     userDisplay.style.display    = 'none';
    }

    // Update gated content areas
    updateGatedContent(user);
  }

  // ── Gated content ───────────────────────────────────────────────────────────
  function updateGatedContent(user) {
    // [data-requires-auth] — show only when logged in
    document.querySelectorAll('[data-requires-auth]').forEach(function (el) {
      el.style.display = user ? '' : 'none';
    });

    // [data-requires-guest] — show only when logged out
    document.querySelectorAll('[data-requires-guest]').forEach(function (el) {
      el.style.display = user ? 'none' : '';
    });

    // [data-requires-admin] — show only for admin role
    document.querySelectorAll('[data-requires-admin]').forEach(function (el) {
      el.style.display = isAdmin(user) ? '' : 'none';
    });

    // Auth-gate panels (full page gate)
    var authGates = document.querySelectorAll('[data-auth-gate]');
    var authContent = document.querySelectorAll('[data-auth-content]');
    var adminGates = document.querySelectorAll('[data-admin-gate]');
    var adminContent = document.querySelectorAll('[data-admin-content]');

    authGates.forEach(function (el) {
      el.style.display = user ? 'none' : '';
    });
    authContent.forEach(function (el) {
      el.style.display = user ? '' : 'none';
    });

    adminGates.forEach(function (el) {
      el.style.display = isAdmin(user) ? 'none' : '';
    });
    adminContent.forEach(function (el) {
      el.style.display = isAdmin(user) ? '' : 'none';
    });
  }

  // ── Button event handlers ───────────────────────────────────────────────────
  if (loginBtn) {
    loginBtn.addEventListener('click', function () {
      identity.open('login');
    });
  }

  if (logoutBtn) {
    logoutBtn.addEventListener('click', function () {
      identity.logout();
    });
  }

  if (mobileLoginBtn) {
    mobileLoginBtn.addEventListener('click', function () {
      identity.open('login');
    });
  }

  if (mobileLogoutBtn) {
    mobileLogoutBtn.addEventListener('click', function () {
      identity.logout();
    });
  }

  // Show login modal from any [data-identity-open] button
  document.querySelectorAll('[data-identity-open]').forEach(function (btn) {
    btn.addEventListener('click', function () {
      identity.open(btn.dataset.identityOpen || 'login');
    });
  });

  document.addEventListener('click', function (e) {
    var signupBtn = e.target.closest('[data-identity-signup-cta]');
    var loginCta = e.target.closest('[data-identity-login-cta]');

    if (signupBtn) {
      identity.open('signup');
    }
    if (loginCta) {
      identity.open('login');
    }
  });

  document.addEventListener('multistep:submitted', function (e) {
    var form = e.target;
    if (!form || !form.matches('[data-identity-signup-on-submit]')) return;

    var result = e.detail && e.detail.result;
    var emailFieldName = form.getAttribute('data-identity-email-field') || 'email';
    var emailField = form.elements[emailFieldName];
    var email = emailField && emailField.value ? emailField.value.trim() : '';
    var emailEls = result ? result.querySelectorAll('[data-registration-email]') : [];
    var guestEls = result ? result.querySelectorAll('[data-registration-guest]') : [];
    var signedInEls = result ? result.querySelectorAll('[data-registration-signed-in]') : [];
    var user = currentUser();

    emailEls.forEach(function (el) {
      el.textContent = email || 'your email address';
    });
    guestEls.forEach(function (el) {
      el.hidden = !!user;
    });
    signedInEls.forEach(function (el) {
      el.hidden = !user;
    });

    if (!user) {
      window.setTimeout(function () {
        identity.open('signup');
      }, 300);
    }
  });

  // ── Identity event hooks ────────────────────────────────────────────────────
  identity.on('init', function (user) {
    updateHeaderUI(user);
  });

  identity.on('login', function (user) {
    updateHeaderUI(user);
    identity.close();
    // Reload so member-only pages rehydrate if needed
    // (light redirect: only on gated pages)
    var redirect = document.body.dataset.authRedirectOnLogin;
    if (redirect) {
      window.location.href = redirect;
    }
  });

  identity.on('logout', function () {
    updateHeaderUI(null);
    var redirect = document.body.dataset.authRedirectOnLogout;
    if (redirect) {
      window.location.href = redirect;
    }
  });

  // ── Initialise ──────────────────────────────────────────────────────────────
  // Explicitly provide the API URL so the widget can resolve /.netlify/identity
  // on custom domains and in local development. Without this the widget fails
  // with "Failed to load settings from /.netlify/identity" on any non-Netlify
  // subdomain URL.
  identity.init({ APIUrl: 'https://ipace-owners.org/.netlify/identity' });

  // Hide all gated content until init fires (avoid flash)
  document.querySelectorAll('[data-auth-content], [data-admin-content]').forEach(function (el) {
    el.style.display = 'none';
  });

})();
