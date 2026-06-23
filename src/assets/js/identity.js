/**
 * identity.js — Firebase Authentication email-link integration.
 *
 * The product UI is passwordless. This script never opens a hosted password
 * widget; it completes Firebase email sign-in links, tracks auth state, adds ID
 * tokens to API submissions, and keeps header/account controls in sync.
 */

(function () {
	'use strict';

	var app = null;
	var auth = null;
	var config = window.ipaceFirebaseConfig;
	window.ipaceIdentityReady = !config;
	window.ipaceIdentityUser = null;

	if (config && window.firebase && window.firebase.initializeApp) {
		app = window.firebase.apps && window.firebase.apps.length
			? window.firebase.app()
			: window.firebase.initializeApp(config);
		auth = app.auth();
		auth.setPersistence(window.firebase.auth.Auth.Persistence.LOCAL).catch(function (err) {
			console.warn('[identity.js] Could not set Firebase persistence.', err);
		});
	} else if (config) {
		console.warn('[identity.js] Firebase SDK not found. Header auth UI disabled.');
	}

	var loginBtn = document.getElementById('identity-login-btn');
	var logoutBtn = document.getElementById('identity-logout-btn');
	var userDisplay = document.getElementById('identity-user-display');
	var mobileLoginBtn = document.getElementById('identity-mobile-login-btn');
	var mobileLogoutBtn = document.getElementById('identity-mobile-logout-btn');
	var pendingEmailLinkUrl = auth && auth.isSignInWithEmailLink(window.location.href)
		? window.location.href
		: '';

	function setVisibility(selector, visible) {
		document.querySelectorAll(selector).forEach(function (el) {
			el.style.display = visible ? '' : 'none';
		});
	}

	function normaliseUser(user) {
		if (!user) return null;
		return {
			uid: user.uid,
			email: user.email || '',
			created_at: user.metadata && user.metadata.creationTime ? user.metadata.creationTime : '',
			firebaseUser: user
		};
	}

	function updateHeaderUI(user) {
		if (user) {
			if (loginBtn) loginBtn.style.display = 'none';
			if (logoutBtn) logoutBtn.style.display = '';
			if (mobileLoginBtn) mobileLoginBtn.style.display = 'none';
			if (mobileLogoutBtn) mobileLogoutBtn.style.display = '';
			setVisibility('[data-requires-auth]', true);
			setVisibility('[data-requires-guest]', false);
			if (userDisplay) {
				userDisplay.style.display = '';
				userDisplay.textContent = user.email || 'Member';
				userDisplay.setAttribute('aria-label', 'My account');
			}
		} else {
			if (loginBtn) loginBtn.style.display = '';
			if (logoutBtn) logoutBtn.style.display = 'none';
			if (mobileLoginBtn) mobileLoginBtn.style.display = '';
			if (mobileLogoutBtn) mobileLogoutBtn.style.display = 'none';
			setVisibility('[data-requires-auth]', false);
			setVisibility('[data-requires-guest]', true);
			if (userDisplay) userDisplay.style.display = 'none';
		}
	}

	function dispatchIdentityState(name, user) {
		var normalised = normaliseUser(user);
		window.ipaceIdentityReady = true;
		window.ipaceIdentityUser = normalised;
		document.dispatchEvent(new CustomEvent(name, {
			detail: { user: normalised }
		}));
	}

	function clearAuthQuery() {
		if (!/(mode=signIn|oobCode=|apiKey=)/.test(window.location.search)) return;
		if (window.history && window.history.replaceState) {
			window.history.replaceState(null, document.title, window.location.pathname + window.location.hash);
		}
	}

	function setAllMagicLinkStatuses(message, tone) {
		document.querySelectorAll('[data-magic-link-status]').forEach(function (status) {
			status.textContent = message;
			status.style.color = tone === 'error' ? 'var(--color-danger)' : 'var(--color-text-muted)';
		});
	}

	function completeEmailLinkIfNeeded() {
		if (!auth || !pendingEmailLinkUrl) {
			return Promise.resolve(false);
		}
		var email = window.localStorage.getItem('ipaceEmailForSignIn') || '';

		function signInWithEmailLink(emailAddress) {
			return auth.signInWithEmailLink(emailAddress, pendingEmailLinkUrl).then(function () {
				window.localStorage.removeItem('ipaceEmailForSignIn');
				pendingEmailLinkUrl = '';
				clearAuthQuery();
				return true;
			});
		}

		email = email.trim();
		if (!email) {
			setAllMagicLinkStatuses('Enter the email address that received this link to finish signing in.', 'info');
			document.querySelectorAll('[data-magic-link-form] input[name="email"]').forEach(function (input) {
				input.focus();
			});
			return Promise.resolve(false);
		}

		return signInWithEmailLink(email).catch(function (err) {
			console.warn('[identity.js] Email-link sign-in failed.', err);
			window.localStorage.removeItem('ipaceEmailForSignIn');
			setAllMagicLinkStatuses('We could not finish sign-in with the remembered email. Enter the email address that received this link to try again.', 'error');
			return false;
		});
	}

	function completePendingEmailLink(email, form, submitBtn) {
		if (!auth || !pendingEmailLinkUrl) return Promise.resolve(false);
		if (submitBtn) submitBtn.disabled = true;
		setMagicLinkStatus(form, 'Completing sign-in...', 'info');
		return auth.signInWithEmailLink(email, pendingEmailLinkUrl).then(function () {
			window.localStorage.removeItem('ipaceEmailForSignIn');
			pendingEmailLinkUrl = '';
			clearAuthQuery();
			setMagicLinkStatus(form, 'Signed in. Loading your account...', 'info');
			return true;
		}).catch(function (err) {
			console.warn('[identity.js] Email-link sign-in failed from form.', err);
			setMagicLinkStatus(form, 'We could not finish sign-in with that link. Check the email address matches the one that received the link, or request a new sign-in link.', 'error');
			return true;
		}).finally(function () {
			if (submitBtn) submitBtn.disabled = false;
		});
	}

	function signOut() {
		if (!auth) return;
		auth.signOut().then(function () {
			var redirect = document.body.dataset.authRedirectOnLogout;
			if (redirect) window.location.href = redirect;
		});
	}

	if (logoutBtn) logoutBtn.addEventListener('click', signOut);
	if (mobileLogoutBtn) mobileLogoutBtn.addEventListener('click', signOut);

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
		if (!auth || !auth.currentUser) return Promise.resolve('');
		return auth.currentUser.getIdToken().catch(function () { return ''; });
	}

	window.ipaceGetIdentityToken = getIdentityToken;

	function setMagicLinkStatus(form, message, tone) {
		var status = form.querySelector('[data-magic-link-status]');
		if (!status) return;
		status.textContent = message;
		status.style.color = tone === 'error' ? 'var(--color-danger)' : 'var(--color-text-muted)';
	}

	function initMagicLinkForms() {
		document.querySelectorAll('[data-magic-link-form]').forEach(function (form) {
			form.addEventListener('submit', function (e) {
				e.preventDefault();
				var emailInput = form.querySelector('input[name="email"]');
				var submitBtn = form.querySelector('button[type="submit"]');
				var email = emailInput ? emailInput.value.trim() : '';

				if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
					setMagicLinkStatus(form, 'Enter a valid email address.', 'error');
					if (emailInput) emailInput.focus();
					return;
				}

				if (pendingEmailLinkUrl) {
					completePendingEmailLink(email, form, submitBtn);
					return;
				}

				if (submitBtn) submitBtn.disabled = true;
				setMagicLinkStatus(form, 'Checking registration and requesting a sign-in link...', 'info');
				window.localStorage.setItem('ipaceEmailForSignIn', email);

				fetch('/api/send-magic-link', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ email: email })
				}).then(function (res) {
					return res.json().catch(function () { return {}; }).then(function (data) {
						if (!res.ok || !data.ok) {
							throw new Error(data && data.error ? data.error : 'Could not send sign-in link');
						}
						setMagicLinkStatus(form, 'If this email address is registered, a secure sign-in link will be sent. You can return here after opening it.', 'info');
					});
				}).catch(function (err) {
					console.warn('[identity.js] Magic link request failed.', err);
					setMagicLinkStatus(form, 'We could not send a sign-in link right now. Please try again or contact us.', 'error');
				}).finally(function () {
					if (submitBtn) submitBtn.disabled = false;
				});
			});
		});
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
			if (token) headers.Authorization = 'Bearer ' + token;

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
					if (email && data.magicLinkSent && !data.signedIn) {
						window.localStorage.setItem('ipaceEmailForSignIn', email);
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

	if (auth) {
		completeEmailLinkIfNeeded().finally(function () {
			auth.onAuthStateChanged(function (user) {
				updateHeaderUI(user);
				dispatchIdentityState(window.ipaceIdentityReady ? (user ? 'identity:login' : 'identity:logout') : 'identity:ready', user);

				if (user) {
					document.querySelectorAll('[data-registration-guest]').forEach(function (el) { el.hidden = true; });
					document.querySelectorAll('[data-registration-signed-in]').forEach(function (el) { el.hidden = false; });
					var redirect = document.body.dataset.authRedirectOnLogin;
					if (redirect) window.location.href = redirect;
				}
			});
		});
	} else {
		updateHeaderUI(null);
		dispatchIdentityState('identity:ready', null);
	}

	initMagicLinkForms();
})();
