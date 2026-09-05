# Identity, Member, and Admin Gating Prompt

Use this prompt when changing Firebase Authentication, passwordless sign-in UI, member
pages, admin pages, or server-verified page gating.

## Goal

Provide passwordless sign-in, sign-out, registration state, member-only pages with live
data, and admin-only pages with live data. All private data access must be verified
server-side by Go Cloud Functions that validate Firebase ID tokens.

## Frontend rules

- Load Firebase Auth from the CDN only when build-time Firebase config is present.
- Do not add a password form or hosted password modal.
- Visible sign-in UI must use custom `[data-magic-link-form]` forms that call
  `POST /api/send-magic-link`. These forms are login-only: the server must send a Firebase
  email link only when the email fingerprint has a Join submission or the exact email belongs
  to an existing Firebase Authentication account. The UI copy must use non-enumerating
  language such as "if this email address is registered".
- `src/assets/js/identity.js` owns email-link completion, header UI, logout, magic-link
  form submission, protected form token injection, and Join result state.
- `src/assets/js/member-auth.js` owns server-side auth verification and data population on
  gated pages.
- Keep JavaScript plain and unbundled.

## identity.js

- Initialise Firebase Auth defensively and expose one `window.ipaceIdentityReadyPromise` that
  resolves only after persistence setup and any incoming email-link completion have finished.
- Complete `signInWithEmailLink` when the user opens a Firebase email link before protected-page
  verification starts; observe subsequent authentication/token changes through Firebase's token
  listener.
- Store and clear `ipaceEmailForSignIn` in `localStorage` for the email-link flow.
- When Join sends a guest registration link, store the submitted email in
  `ipaceEmailForSignIn` so the clicked link can complete without asking again in the same
  browser.
- Do not use `window.prompt` for email-link completion. If the stored email is missing, use
  the visible `[data-magic-link-form]` and `[data-magic-link-status]` UI to ask for the email
  address that received the link, then complete the pending link from that form submission. If
  Firebase rejects a link, clear only the pending link state and auth query parameters without
  reloading; the same visible form must then request a fresh sign-in link rather than retrying
  the rejected action code.
- Generate one opaque, session-scoped sign-in support code (for example `IP-ABCD-2345`).
  It must contain no email address, token, UID, or other personal data. Attach it as
  `X-Ipace-Auth-Trace` to every authenticated API request, magic-link request, and gate
  verification. Carry a valid code through the passwordless email-link `continueUrl` so a
  new tab or browser hand-off remains one support journey. On a recoverable sign-in failure,
  show the member the code and ask them to report it with the approximate time; do not show
  it during a successful sign-in.
- Send only bounded lifecycle names and outcomes to `POST /api/auth-diagnostics` for magic-link
  requesting/completion and member/admin verification. Require both an allowed browser `Origin`
  and an `X-Ipace-Auth-Trace` header exactly matching the body code. Never send an email address,
  Firebase token, exception text, browser URL, or free-form diagnostic data in this telemetry.
- Expose `window.ipaceGetIdentityToken()` so form/API code can attach
  `Authorization: Bearer <Firebase ID token>`.
- Update header and mobile controls based on current user state.
- On member routes, render a compact, single-line secondary breadcrumb row in the shared
  header (`Member › current page`). Emphasise the current crumb with `aria-current="page"`
  and a visible active style. Keep the member page title compact and do not repeat a
  redundant `Member` eyebrow immediately below the breadcrumb.
- Apply the same pattern to every admin route (`Admin › current page`). Link the `Admin`
  section crumb back to `/admin/` from tool pages, render it as text on the dashboard,
  emphasise the current crumb, and remove the redundant `Admin` page eyebrow.
- Keep the mobile drawer's labelled Member section visible and colour-contrast compliant:
  guests see Sign in; authenticated members see My Data, Add Vehicle and Sign out. Administrators
  additionally see one claim-gated Admin action.
- The signed-out mobile header Sign in control must be visible only below the mobile
  breakpoint. Its hiding rule must outrank the shared button display rule so desktop never
  renders both the desktop and mobile Sign in actions.
- Show public `Join` CTA only to guests. Signed-in users must have exactly one obvious
  `My Data` route to the `/member/account/` homepage in desktop and mobile navigation.
  Do not expose the signed-in email address as a separate header action.
- Keep authenticated account and vehicle-registration templates within `src/member/`, at
  `/member/account/` and `/member/submit-vehicle-data/`. Permanently redirect their former
  top-level routes so bookmarks and previously issued links remain usable.
- Support `[data-magic-link-form]` with `[data-magic-link-status]`.
- Render the shared pending and passwordless sign-in states through the configurable
  `partials/auth-login-gate.njk` macro; do not duplicate login-gate markup across member and
  admin pages. Keep authenticated content and server-side authorisation page-specific.
- Handle `multistep:submitted` + `data-database-submit`.
- Do not reveal gated content directly. Gated content is revealed only by
  `member-auth.js` after an API returns 200.

## member-auth.js

- On page load, find `[data-auth-container]` and `[data-admin-container]`, but wait for the shared
  identity-ready promise before making a protected API request. Do not use a speculative timer that
  sends an unauthenticated request while Firebase is still restoring a session.
- Fetch:
  - member pages: `GET /api/member-data`
  - admin pages: `GET /api/admin-data`
- Send the Firebase ID token in `Authorization: Bearer <token>`.
- On 200: hide the gate, show content, populate data from response.
- On 401: for a known signed-in member or administrator, force-refresh the Firebase ID token and
  retry once before showing the login gate. For a genuinely signed-out user, show the login gate.
- On a network failure or 5xx response, keep protected member or administrator content hidden but
  show a recoverable sign-in-verification error with a `Try again` control; do not send the user
  back through the magic-link flow. Serialise both member and administrator verification so
  duplicate lifecycle events cannot race each other.
- If a known signed-in member receives a 401 even after the one forced token refresh, show the
  recoverable verification error and `Try again` control rather than putting them back into the
  magic-link gate. This avoids a transient token/session race becoming a sign-in loop. Include
  the session support code in that error. Retain the distinct access-restricted UI for an
  administrator receiving 403.
- On 403 for admin: show access-restricted gate.
- Populate vehicle lists, join info, account preferences, admin stats, join table, and
  vehicle table from the API response.
- Account preferences must display the latest saved Join consent and membership state from
  the member snapshot, including contact permission, anonymised aggregate-analysis consent,
  participation acknowledgement, relationship, and volunteering interests where present.
  Do not show placeholder copy saying preferences will be manageable in a future release.
  Preference editing needs a server-side audited account update flow before controls are
  shown as editable.

## Server-side APIs

The shared desktop and mobile navigation exposes exactly one `Admin` action, linking to
`/admin/`, after Firebase claims indicate admin access. The admin dashboard is the directory for
Review Queue, Facebook Assistant, Email Campaigns and Instagram Campaigns; do not duplicate those
tool links in the shared header or mobile drawer. Each admin page keeps the dashboard reachable
through its compact breadcrumb. Campaign APIs independently verify the ID token
and admin role server-side.

| Function | Auth Required | Purpose |
|---|---|---|
| `MemberData` | Firebase user | Return the authenticated user's private snapshot. |
| `AdminData` | Firebase admin custom claim | Return admin review data. |
| `AuthDiagnostics` | Public, same-origin | Record bounded, PII-free passwordless lifecycle events keyed by the opaque support code. |

### Authorization tracing matrix

For a reported support code, Cloud Logging must show every authorization decision made by the
server. The common guard writes the sanitized route, required role, decision, and HTTP status
only when a valid `X-Ipace-Auth-Trace` is present; it must not log personal data, token data,
request bodies, or free-form browser errors.

| Request class | Required role | Central guard | Decisions recorded |
|---|---|---|---|
| Member reads, exports, survey responses, and member-owned writes | Firebase user | `requireUser` | `allowed`, `missing-token`, `invalid-token` (200/401) |
| Admin data, statistics, survey management/results, campaigns, and publishing tools | Firebase admin claim | `requireAdmin` / `campaignAuthorize` | member decision followed by admin `allowed` or `admin-claim-missing` (200/401/403) |
| Passwordless start and lifecycle diagnostics | No role; same-origin only | public handler + origin validation | lifecycle events only, never an authorization claim |

The browser must also report bounded `persistence` and `identity-observer` lifecycle outcomes,
alongside magic-link and gate verification outcomes. This makes it possible to reconstruct the
journey around each authorization event without relying on client-side UI gating as security.

Admin access is granted through Firebase Auth custom claims: `admin: true` or
`roles: ["admin"]`.

OpenTofu owns the authoritative administrator set through its tested Identity Platform
reconciliation bridge because the Google provider exposes no Firebase Auth user data source.
Always include `dan@kanzi.co.uk` as a required administrator, resolve configured emails to the
environment-specific Firebase UID during apply, preserve unrelated claims, and remove only
admin access from users removed from configuration. Fail when a configured user does not exist.
After sign-in, inspect the Firebase ID-token result and expose the desktop/mobile Admin action
only for `admin: true` or a `roles` entry containing `admin`. This is a discoverability aid only;
the destination must continue to require server-side `AdminData` verification.

## Page pattern

Login gates are visible by default or via a pending state. Content must have `hidden` until
the API confirms auth.

```njk
<div data-auth-container>
  <div class="auth-gate" data-auth-pending>Checking sign-in...</div>
  <div class="auth-gate" data-auth-login-gate hidden>...</div>
  <div class="auth-content" data-auth-content hidden>...</div>
</div>
```

## Magic-link flow

- Join completion makes exactly one browser request: `POST /api/submit-join`.
- `SubmitJoin` stores the Join answers and sends a Firebase email link for guests.
- `SendMagicLink` remains available for existing users who need another link. Treat either
  a matching Join submission or an existing Firebase Authentication account as registered;
  this supports claim-managed staging administrators whose preview data does not include a
  copied Join record.
- Both endpoints must avoid account enumeration.
- Do not call Firebase Identity Toolkit directly from browser code except through the
  Firebase Auth SDK's email-link completion.

## Validation

- Run `make build`.
- Run `make test` for frontend/auth wiring changes.
- Run `GOCACHE=/tmp/ipace-owners-go-build make test-go` or `go test ./...` in
  `functions/firebase-go` for backend/auth changes.
- Test logged-out, logged-in, and admin states.
- Confirm no private data appears in `_site/`.
