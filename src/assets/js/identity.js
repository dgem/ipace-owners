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
    console.warn('[identity.js] netlify-identity-widget not found. Header auth UI disabled; magic-link form handoff remains available.');
  }

  // ── Helper: get current user ────────────────────────────────────────────────
  function currentUser() {
    if (!identity) return null;
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
  if (identity && loginBtn) {
    loginBtn.addEventListener('click', function () {
      identity.open('login');
    });
  }

  if (identity && logoutBtn) {
    logoutBtn.addEventListener('click', function () {
      identity.logout();
    });
  }

  if (identity && mobileLoginBtn) {
    mobileLoginBtn.addEventListener('click', function () {
      identity.open('login');
    });
  }

  if (identity && mobileLogoutBtn) {
    mobileLogoutBtn.addEventListener('click', function () {
      identity.logout();
    });
  }

  // Show login modal from any [data-identity-open] button
  if (identity) {
    document.querySelectorAll('[data-identity-open]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        identity.open(btn.dataset.identityOpen || 'login');
      });
    });
  }

  function setResultVisibility(result, selector, visible) {
    if (!result) return;
    result.querySelectorAll(selector).forEach(function (el) {
      el.hidden = !visible;
    });
  }

  function showNetlifyFormSaved(result) {
    setResultVisibility(result, '[data-registration-form-saved]', true);
    setResultVisibility(result, '[data-registration-form-error]', false);
  }

  function showNetlifyFormError(result) {
    setResultVisibility(result, '[data-registration-form-saved]', false);
    setResultVisibility(result, '[data-registration-form-error]', true);
  }

  function submitNetlifyForm(form, result) {
    if (!form || !form.matches('[data-netlify-form-submit]')) return;

    var formName = form.getAttribute('name');
    var action = form.getAttribute('action') || window.location.pathname;
    var formData = new FormData(form);

    if (formName && !formData.has('form-name')) {
      formData.append('form-name', formName);
    }

    fetch(action, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams(formData).toString()
    }).then(function (res) {
      if (!res.ok) {
        throw new Error('Netlify Forms submission failed with status ' + res.status);
      }
      showNetlifyFormSaved(result);
    }).catch(function (err) {
      console.warn('[identity.js] Netlify Forms submission failed.', err);
      showNetlifyFormError(result);
    });
  }

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

    submitNetlifyForm(form, result);

    if (user) {
      guestEls.forEach(function (el) { el.hidden = true; });
      signedInEls.forEach(function (el) { el.hidden = false; });
      return;
    }

    // Not signed in — reveal guest section immediately, then send magic link.
    guestEls.forEach(function (el) { el.hidden = false; });
    signedInEls.forEach(function (el) { el.hidden = true; });

    if (!email) return;

    var nameField = form.elements['name'];
    var name = nameField && nameField.value ? nameField.value.trim() : '';

    function showLinkSent() {
      setResultVisibility(result, '[data-registration-link-sent]', true);
      setResultVisibility(result, '[data-registration-error]', false);
    }

    function showError() {
      setResultVisibility(result, '[data-registration-link-sent]', false);
      setResultVisibility(result, '[data-registration-error]', true);
    }

    // Valid Identity failures return HTTP 200 for enumeration resistance, so
    // read the JSON body's `ok` flag instead of relying on res.ok alone.
    fetch('/.netlify/functions/send-magic-link', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email, name: name })
    }).then(function (res) {
      return res.json();
    }).then(function (data) {
      if (data && data.ok) { showLinkSent(); } else { showError(); }
    }).catch(showError);
  });

  // ── Identity event hooks ────────────────────────────────────────────────────
  if (identity) {
    identity.on('init', function (user) {
      updateHeaderUI(user);
    });

    identity.on('login', function (user) {
      updateHeaderUI(user);
      identity.close();

      // If the join result panel is visible, flip it to the signed-in state so
      // the guest CTAs ("check your inbox") are hidden after the user logs in.
      var guestEls = document.querySelectorAll('[data-registration-guest]');
      var signedInEls = document.querySelectorAll('[data-registration-signed-in]');
      if (guestEls.length || signedInEls.length) {
        guestEls.forEach(function (el) { el.hidden = true; });
        signedInEls.forEach(function (el) { el.hidden = false; });
      }

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
    // Derive the API URL from the current origin so the widget resolves
    // /.netlify/identity correctly in all environments: production, the custom
    // domain, and deploy previews. Using a hard-coded production URL would break
    // deploy previews (cross-origin CSP) and point all test signups at production.
    identity.init({ APIUrl: window.location.origin + '/.netlify/identity' });
  }

  // Hide all gated content until init fires (avoid flash)
  document.querySelectorAll('[data-auth-content], [data-admin-content]').forEach(function (el) {
    el.style.display = 'none';
  });

})();
