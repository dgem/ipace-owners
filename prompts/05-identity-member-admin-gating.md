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
- If the stored email is missing or rejected by Firebase, prompt the user to confirm the
  email address that received the link before falling back to the logged-out UI. Surface a
  visible error in `[data-magic-link-status]` if completion still fails.
- Expose `window.ipaceGetIdentityToken()` so form/API code can attach
  `Authorization: Bearer <Firebase ID token>`.
- Update header and mobile controls based on current user state.
- Show public `Join` CTA only to guests. Signed-in users must have exactly one obvious
  `My Data` route to `/member/dashboard/` in desktop and mobile navigation. The signed-in
  email address should link to `/account/` as the account-management route.
- Support `[data-magic-link-form]` with `[data-magic-link-status]`.
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
- Populate vehicle lists, join info, admin stats, join table, and vehicle table from the
  API response.

## Server-side APIs

| Function | Auth Required | Purpose |
|---|---|---|
| `MemberData` | Firebase user | Return the authenticated user's private snapshot. |
| `AdminData` | Firebase admin custom claim | Return admin review data. |

Admin access is granted through Firebase Auth custom claims: `admin: true` or
`roles: ["admin"]`.

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

- Run `npm run build`.
- Run `npm test` for frontend/auth wiring changes.
- Run `go test ./...` in `functions/firebase-go` for backend/auth changes.
- Test logged-out, logged-in, and admin states.
- Confirm no private data appears in `_site/`.
