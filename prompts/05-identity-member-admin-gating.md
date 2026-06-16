# Identity, Member, and Admin Gating Prompt

Implement or refine Netlify Identity integration and server-side gated pages.

## Goal

Provide frontend Identity UX for sign in, sign out, registration, member-only pages with live data, and admin-only pages with live data — all verified server-side via Netlify Functions that check Identity JWTs.

**Client-side gating has been removed.** All data access is now verified server-side. The `member-auth.js` script fetches from `member-data` or `admin-data` Netlify Functions on page load; only the Function response determines whether content is shown.

## Requirements

- Load `netlify-identity-widget` from `https://identity.netlify.com/v1/netlify-identity-widget.js` in the base layout only as the current Identity session/JWT adapter while that dependency remains.
- Do not use the Netlify Identity modal UI. The site is passwordless: visible sign-in UI must use custom email magic-link forms that call `POST /.netlify/functions/send-magic-link`.
- Use `src/assets/js/identity.js` for header UI, logout, magic-link form submission, database form token injection, and Join result state.
- Use `src/assets/js/member-auth.js` for server-side auth verification and data population on gated pages.
- Keep JavaScript plain and unbundled.

### identity.js — Header UI only

- Initialize `window.netlifyIdentity` defensively.
- Update header and mobile controls based on current user state.
- In the main and mobile navigation, show the public `Join` CTA only to guests. For
  signed-in users, replace it with an `Account` or `My Data` link so members can reach
  their account page without returning to the Join flow.
- Do not add buttons with `data-identity-open`, and do not call `identity.open(...)`.
- Support `[data-magic-link-form]` forms with an email input and `[data-magic-link-status]`
  status copy. These forms call `send-magic-link` and show a check-your-email message.
- Handle form submission via `multistep:submitted` + `data-database-submit`.
- **No client-side content gating.** `identity.js` does not show/hide gated content areas.

### member-auth.js — Server-side auth verification

- On page load, find `[data-auth-container]` (member pages) or `[data-admin-container]` (admin pages).
- Fetch the appropriate Netlify Function:
    - Member pages → `GET /.netlify/functions/member-data`
    - Admin pages → `GET /.netlify/functions/admin-data`
- Send the Identity JWT in the `Authorization: Bearer <token>` header. Do not rely on
  browser cookies alone for server Function auth.
- On 200: hide the login gate, show content, populate data from response.
- On 401: keep login gate visible (not authenticated).
- On 403 (admin only): show "access restricted" gate (authenticated but not admin).
- Populate `[data-vehicle-list]`, `[data-join-info]`, `[data-admin-stats]`, `[data-admin-join-table]`, `[data-admin-vehicle-table]` from Function response.

### Page templates — server-side auth pattern

**Member-gated pages** (dashboard, account):
```njk
<div data-auth-container>
  <div class="auth-gate" data-auth-login-gate>
    <!-- Passwordless magic-link prompt — shown by default -->
  </div>
  <div class="auth-content" data-auth-content hidden>
    <!-- Content only revealed after server confirms auth -->
    <div data-vehicle-container>
      <div data-vehicle-list></div>
    </div>
    <div data-join-container></div>
  </div>
</div>
```

**Admin-gated pages** (review queue):
```njk
<div data-admin-container>
  <div class="auth-gate" data-auth-login-gate>
    <!-- Passwordless magic-link prompt — shown by default -->
  </div>
  <div class="auth-gate" data-admin-only-gate hidden>
    <!-- "Access restricted" — shown when authenticated but not admin -->
  </div>
  <div class="auth-content" data-admin-content hidden>
    <!-- Content only revealed after server confirms admin role -->
    <div data-stats-container></div>
    <div data-join-table-container></div>
    <div data-vehicle-table-container></div>
  </div>
</div>
```

## Server-side Functions

| Function | Auth Required | What it does |
|---|---|---|
| `member-data.js` | Yes (signed-in user) | Returns the authenticated user's own join and vehicle-basics records. Verifies `context.clientContext.user.sub`. |
| `admin-data.js` | Yes (admin role) | Returns all join and vehicle-basics records. Verifies JWT + `app_metadata.roles` contains `admin`. |

### member-data.js

- GET only, CORS preflight support.
- Requires `context.clientContext.user` — returns 401 if missing.
- Lists `join/` records filtered by `identityUserId === user.sub`.
- Reads the authenticated user's private member/account snapshot, which may include
  multiple vehicle records.
- Returns `{ identityUserId, email, joinRecords[], vehicleRecords[] }`.
- Does **not** expose `review` data to members.

### admin-data.js

- GET only, CORS preflight support.
- Requires `context.clientContext.user` — returns 401 if missing.
- Requires `app_metadata.roles` contains `admin` — returns 403 if not admin.
- Lists **all** `join/` records (no user filter).
- Reads review-capable member, Join, vehicle, and vehicle-basics records from Postgres.
- Returns `{ identityUserId, email, joinRecords[], vehicleRecords[] }`.
- **Does** expose `review` data and `userEmailHash` to admins.

## Pages

- Member dashboard at `/member/dashboard/` — shows user's vehicle records and join info.
- Account page at `/account/` — shows passwordless sign-in when logged out, and account details plus vehicle submissions when signed in.
- Submit vehicle data at `/submit-vehicle-data/` — server-gated form page.
- Admin review queue at `/admin/review-queue/` — shows all submissions with stats and tables.

## Magic Link Flow

On Join form submit, `identity.js` must make exactly one browser request:
`POST /.netlify/functions/submit-join`.

`submit-join` stores the Join answers and then calls shared server-side magic-link
code for logged-out users.

`send-magic-link` remains available as a standalone endpoint for users who already
have an account and need another sign-in link, but Join completion must not call it
separately from the browser.

The shared server-side magic-link code calls Netlify Identity internally (signup for new users, magiclink for
existing ones). For valid same-origin POST requests, it returns HTTP 200 with an
`ok` flag rather than exposing whether the email was new or existing, preventing
account enumeration. It may still return non-200 responses for invalid JSON, invalid
email input, unsupported methods, or disallowed cross-origin requests.

The function must reject disallowed browser origins before calling Netlify Identity,
including `no-cors` style cross-site requests that could otherwise trigger email sends
even though the response would be unreadable to the attacker.

On success, show a "check your inbox" message — no signup/login modal is opened.

Do not call the Netlify Identity REST API directly from the browser. Use the function.

Do not open `identity.open('signup')` or `identity.open('login')` anywhere in the product UI.

When `identity.on('login')` fires after the join result is visible, hide the guest
section (`data-registration-guest`) and show the signed-in message
(`data-registration-signed-in`) so the UI reflects the new auth state.

Netlify Identity must have **autoconfirm disabled** for the confirmation email
(magic link) to be sent. Verify under Site Settings → Identity → Registration.

`identity.init()` must be called with `{ APIUrl: window.location.origin + '/.netlify/identity' }`
so that the widget resolves settings correctly in all environments (same-origin, no
hard-coded domain).

The browser must read `submit-join`'s JSON response for both storage and `magicLinkSent`
state. Do not fire a second request to `send-magic-link` from Join completion.

Register the Join form `multistep:submitted` handler even if the Netlify Identity session
adapter is unavailable or blocked, so `submit-join` is still called. Widget absence should
disable only session-specific behavior such as logout and JWT-backed submissions.

Document that local testing of Identity email Functions requires `npm run dev` running
Netlify Dev, or a deploy preview; the plain Eleventy dev server will not serve Functions.

Document that Netlify Dev serves Functions but not Netlify Identity. Local magic-link
testing must set `NETLIFY_IDENTITY_BASE_URL` to the deployed site's
`https://.../.netlify/identity` endpoint; do not fall back to localhost for Identity.

## Security

**All data access is server-side verified.** The browser cannot access private data without a valid Identity JWT. `member-auth.js` relies entirely on Function responses — if the Function rejects the request, the gate stays visible.

## Validation

- Run `npm run build`.
- Run `npm test` when changing Functions or auth behavior.
- Test logged-out states — gates should be visible.
- Test logged-in states — member pages should show user's own data.
- Test admin states — admin pages should show all data only for admin role.
- Confirm no real private data is exposed in static output.
