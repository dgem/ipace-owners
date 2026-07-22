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
  email link only when the email fingerprint already has a Join submission, and the UI copy
  must use non-enumerating language such as "if this email address is registered".
- `src/assets/js/identity.js` owns email-link completion, header UI, logout, magic-link
  form submission, protected form token injection, and Join result state.
- `src/assets/js/member-auth.js` owns server-side auth verification and data population on
  gated pages.
- Keep JavaScript plain and unbundled.

## identity.js

- Initialise Firebase Auth defensively.
- Complete `signInWithEmailLink` when the user opens a Firebase email link.
- Store and clear `ipaceEmailForSignIn` in `localStorage` for the email-link flow.
- When Join sends a guest registration link, store the submitted email in
  `ipaceEmailForSignIn` so the clicked link can complete without asking again in the same
  browser.
- Do not use `window.prompt` for email-link completion. If the stored email is missing or
  rejected by Firebase, use the visible `[data-magic-link-form]` and
  `[data-magic-link-status]` UI to ask for the email address that received the link, then
  complete the pending link from that form submission.
- Expose `window.ipaceGetIdentityToken()` so form/API code can attach
  `Authorization: Bearer <Firebase ID token>`.
- Update header and mobile controls based on current user state.
- Show public `Join` CTA only to guests. Signed-in users must have exactly one obvious
  `My Data` route to `/member/dashboard/` in desktop and mobile navigation. The signed-in
  email address should link to `/member/account/` as the account-management route.
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

- On page load, find `[data-auth-container]` and `[data-admin-container]`.
- Fetch:
  - member pages: `GET /api/member-data`
  - admin pages: `GET /api/admin-data`
- Send the Firebase ID token in `Authorization: Bearer <token>`.
- On 200: hide the gate, show content, populate data from response.
- On 401: keep login gate visible.
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

The complete admin navigation, including `/admin/email-campaigns/`, lives in a claim-gated
secondary row of the desktop site header, aligned right, and in a labelled section of the mobile
drawer. Do not repeat it inside individual admin page content. It remains hidden until Firebase
claims indicate admin access, while campaign APIs independently verify the ID token and admin
role server-side.

| Function | Auth Required | Purpose |
|---|---|---|
| `MemberData` | Firebase user | Return the authenticated user's private snapshot. |
| `AdminData` | Firebase admin custom claim | Return admin review data. |

Admin access is granted through Firebase Auth custom claims: `admin: true` or
`roles: ["admin"]`.

OpenTofu owns the authoritative administrator set through its tested Identity Platform
reconciliation bridge because the Google provider exposes no Firebase Auth user data source.
Always include `dan@kanzi.co.uk` as a required administrator, resolve configured emails to the
environment-specific Firebase UID during apply, preserve unrelated claims, and remove only
admin access from users removed from configuration. Fail when a configured user does not exist.
After sign-in, inspect the Firebase ID-token result and expose desktop/mobile admin navigation
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
- `SendMagicLink` remains available for existing users who need another link.
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
