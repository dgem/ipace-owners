# Identity, Member, and Admin Gating Prompt

Implement or refine Netlify Identity integration and placeholder gated pages.

## Goal

Provide frontend Identity UX for sign in, sign out, registration, member-only placeholder pages, and admin-only placeholder pages while making clear that real data access requires server-side JWT verification later.

## Requirements

- Load `netlify-identity-widget` from `https://identity.netlify.com/v1/netlify-identity-widget.js` in the base layout.
- Use `src/assets/js/identity.js` for Identity behavior.
- Keep JavaScript plain and unbundled.
- Initialize `window.netlifyIdentity` defensively.
- Update header and mobile controls based on current user state.
- Support buttons with `data-identity-open`.
- Support member UI attributes:
  - `data-auth-gate`
  - `data-auth-content`
  - `data-requires-auth`
  - `data-requires-guest`
- Support admin UI attributes:
  - `data-admin-gate`
  - `data-admin-content`
  - `data-requires-admin`
- Treat admin users as users with `admin` in `user.app_metadata.roles`.
- Support optional body redirects:
  - `data-auth-redirect-on-login`
  - `data-auth-redirect-on-logout`

## Pages

- Member dashboard at `/member/dashboard/`.
- Account page at `/account/`.
- Admin review queue at `/admin/review-queue/`.
- Header controls for login, signup/register where relevant, and logout.
- Sign-in buttons should open the Identity modal without custom password storage.
- Join or registration completion uses a **magic link flow** via a server-side Netlify Function:
  - On Join form submit, `identity.js` must make exactly one browser request:
    `POST /.netlify/functions/submit-join`.
  - `submit-join` stores the Join answers and then calls shared server-side magic-link
    code for logged-out users.
  - `send-magic-link` remains available as a standalone endpoint for users who already
    have an account and need another sign-in link, but Join completion must not call it
    separately from the browser.
  - The shared server-side magic-link code calls Netlify Identity internally (signup for new users, recover for
    existing ones). For valid same-origin POST requests, it returns HTTP 200 with an
    `ok` flag rather than exposing whether the email was new or existing, preventing
    account enumeration. It may still return non-200 responses for invalid JSON, invalid
    email input, unsupported methods, or disallowed cross-origin requests.
  - The function must reject disallowed browser origins before calling Netlify Identity,
    including `no-cors` style cross-site requests that could otherwise trigger email sends
    even though the response would be unreadable to the attacker.
  - On success, show a "check your inbox" message — no signup modal is opened.
  - Do not call the Netlify Identity REST API directly from the browser. Use the function.
- Do not open `identity.open('signup')` or `identity.open('login')` on join form completion.
- When `identity.on('login')` fires after the join result is visible, hide the guest
  section (`data-registration-guest`) and show the signed-in message
  (`data-registration-signed-in`) so the UI reflects the new auth state.
- Netlify Identity must have **autoconfirm disabled** for the confirmation email
  (magic link) to be sent. Verify under Site Settings → Identity → Registration.
- `identity.init()` must be called with `{ APIUrl: window.location.origin + '/.netlify/identity' }`
  so that the widget resolves settings correctly in all environments (same-origin, no
  hard-coded domain).
- The browser must read `submit-join`'s JSON response for both storage and `magicLinkSent`
  state. Do not fire a second request to `send-magic-link` from Join completion.
- Register the Join form `multistep:submitted` handler even if the Netlify Identity widget
  is unavailable or blocked, so `submit-join` is still called. Widget absence should disable
  only widget-specific UI such as modal open/logout controls.
- Document that local testing of Identity email Functions requires `npm run dev` running
  Netlify Dev, or a deploy preview; the plain Eleventy dev server will not serve Functions.
- Document that Netlify Dev serves Functions but not Netlify Identity. Local magic-link
  testing must set `NETLIFY_IDENTITY_BASE_URL` to the deployed site's
  `https://.../.netlify/identity` endpoint; do not fall back to localhost for Identity.

## Security copy

Every placeholder member/admin area that refers to future private data must say that frontend gating is not access control. Future Netlify Functions must verify Identity JWTs server-side and check roles before returning sensitive data.

## Validation

- Run `npm run build`.
- Run `npm test` when changing Identity Function handoff or Join completion behavior.
- Test logged-out states.
- Confirm gated content is hidden until Identity initializes.
- Confirm logged-in/member placeholders respond to Identity state.
- Confirm admin placeholders check for an admin role where possible.
- Confirm placeholder pages do not expose real private data.
