/**
 * identity.js — Netlify Identity UI integration.
 *
 * Initialises netlify-identity-widget and keeps the header UI in sync
 * with the current auth state. Handles login/logout buttons, form submission
 * token injection, and registration state display.
 *
 * NOTE: Client-side gating has been removed. All data access is now verified
 * server-side via member-auth.js which calls Netlify Functions that check
 * Identity JWTs. See member-auth.js for page-level auth verification.
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

	// ── Header UI update ────────────────────────────────────────────────────────
	var loginBtn         = document.getElementById('identity-login-btn');
	var logoutBtn        = document.getElementById('identity-logout-btn');
	var userDisplay      = document.getElementById('identity-user-display');
	var mobileLoginBtn   = document.getElementById('identity-mobile-login-btn');
	var mobileLogoutBtn  = document.getElementById('identity-mobile-logout-btn');

	function updateHeaderUI(user) {
		if (user) {
			// Logged in
			if (loginBtn)        loginBtn.style.display         = 'none';
			if (logoutBtn)       logoutBtn.style.display        = '';
			if (mobileLoginBtn)  mobileLoginBtn.style.display   = 'none';
			if (mobileLogoutBtn) mobileLogoutBtn.style.display  = '';
			if (userDisplay) {
				userDisplay.style.display    = '';
				userDisplay.textContent      = user.email || 'Member';
			}
		} else {
			// Logged out
			if (loginBtn)        loginBtn.style.display         = '';
			if (logoutBtn)       logoutBtn.style.display        = 'none';
			if (mobileLoginBtn)  mobileLoginBtn.style.display   = '';
			if (mobileLogoutBtn) mobileLogoutBtn.style.display  = 'none';
			if (userDisplay)     userDisplay.style.display      = 'none';
		}
	}

	// NOTE: Client-side gated content (data-auth-gate, data-admin-gate, etc.)
	// has been removed. All data access is now verified server-side via
	// member-auth.js calling Netlify Functions with JWT verification.

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

	function setSubmissionId(result, id) {
		if (!result || !id) return;
		result.querySelectorAll('[data-database-submission-id]').forEach(function (el) {
			el.textContent = id;
		});
	}

	function showDatabaseSaved(result, id) {
		setSubmissionId(result, id);
		setResultVisibility(result, '[data-database-success]', true);
		setResultVisibility(result, '[data-database-error]', false);
		setResultVisibility(result, '[data-database-auth-error]', false);
	}

	function showDatabaseError(result) {
		setResultVisibility(result, '[data-database-success]', false);
		setResultVisibility(result, '[data-database-error]', true);
		setResultVisibility(result, '[data-database-auth-error]', false);
	}

	function showDatabaseAuthError(result) {
		setResultVisibility(result, '[data-database-success]', false);
		setResultVisibility(result, '[data-database-error]', false);
		setResultVisibility(result, '[data-database-auth-error]', true);
	}

	function showRegistrationState(result, data) {
		if (!result || !data || !('magicLinkSent' in data)) return;

		setResultVisibility(result, '[data-registration-guest]', !data.signedIn);
		setResultVisibility(result, '[data-registration-signed-in]', !!data.signedIn);
		setResultVisibility(result, '[data-registration-link-sent]', !!data.magicLinkSent && !data.signedIn);
		setResultVisibility(result, '[data-registration-error]', !data.magicLinkSent && !data.signedIn);
	}

	function formDataToObject(formData) {
		var payload = {};
		formData.forEach(function (value, key) {
			if (key === 'bot-field') return;
			if (Object.prototype.hasOwnProperty.call(payload, key)) {
				if (!Array.isArray(payload[key])) payload[key] = [payload[key]];
				payload[key].push(value);
			} else {
				payload[key] = value;
			}
		});
		return payload;
	}

	function getIdentityToken() {
		var user = currentUser();
		if (!user || typeof user.jwt !== 'function') return Promise.resolve('');
		return user.jwt().catch(function () { return ''; });
	}

	function submitDatabaseForm(form, result) {
		if (!form || !form.matches('[data-database-submit]')) return;

		var endpoint = form.getAttribute('data-database-submit') || form.getAttribute('action');
		var requiresAuth = form.hasAttribute('data-database-requires-auth');
		var formData = new FormData(form);
		var payload = formDataToObject(formData);
		var email = payload.email || '';

		if (result && email) {
			result.querySelectorAll('[data-registration-email]').forEach(function (el) {
				el.textContent = email;
			});
		}

		getIdentityToken().then(function (token) {
			if (requiresAuth && !token) {
				showDatabaseAuthError(result);
				return;
			}

			var headers = { 'Content-Type': 'application/json' };
			if (token) {
				headers.Authorization = 'Bearer ' + token;
			}

			return fetch(endpoint, {
				method: 'POST',
				headers: headers,
				body: JSON.stringify(payload)
			}).then(function (res) {
				if (res.status === 401) {
					showDatabaseAuthError(result);
					return null;
				}
				return res.json().then(function (data) {
					if (!res.ok || !data || !data.ok) {
						throw new Error(data && data.error ? data.error : 'Database submission failed');
					}
					showDatabaseSaved(result, data.id);
					showRegistrationState(result, data);
					return data;
				});
			});
		}).catch(function (err) {
			console.warn('[identity.js] Database submission failed.', err);
			showDatabaseError(result);
		});
	}

	document.addEventListener('multistep:submitted', function (e) {
		var form = e.target;
		if (!form || !form.matches('[data-database-submit]')) return;
		submitDatabaseForm(form, e.detail && e.detail.result);
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

})();
