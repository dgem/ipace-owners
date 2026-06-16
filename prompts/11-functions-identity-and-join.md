# Identity Magic Link and Join Functions Prompt

Use this prompt when changing:

- `functions/firebase-go` magic-link or Join handlers
- Join form submission wiring in `src/join.njk`
- `src/assets/js/identity.js`

## Goal

Maintain a single browser submission for joining the group while preserving a standalone
magic-link request path for existing users.

## Current flow

- The Join form posts once to `POST /api/submit-join`.
- `SubmitJoin` validates required Join fields and consent.
- `SubmitJoin` stores the Join record in Firestore.
- For logged-out users, `SubmitJoin` sends a Firebase email sign-in link in the same
  request.
- For signed-in users, `SubmitJoin` stores the record and regenerates the member snapshot.
- `SendMagicLink` remains available at `POST /api/send-magic-link` for users who need a
  sign-in link without resubmitting Join.

## Magic-link rules

- Do not make Join completion fire a second browser request to `SendMagicLink`.
- Custom passwordless sign-in forms outside Join may call `SendMagicLink`.
- Do not add a password login UI.
- `SendMagicLink` calls Firebase Identity Toolkit server-side and returns generic success
  for syntactically valid requests. Do not reveal whether the email address is registered.
- Reject disallowed origins before triggering email side effects.
- Log only email fingerprints and provider status/error summaries, never raw email bodies
  or tokens.

## Join record shape

Join records include:

- `id`, `type`, `createdAt`, `updatedAt`.
- optional `identityUserId`.
- `contact`: name, email, country.
- `membership`: one combined relationship/ownership value plus skills.
- `consents`: contact, not-legal-claim acknowledgement, anonymised analysis.
- `review`: status and verification level.

Accepted relationship values:

- `current-owner-one`
- `current-owner-multiple`
- `former-owner`
- `prospective-buyer`
- `helping-owner`
- `trade-specialist`
- `other`

## Browser result state

`identity.js` should read the JSON response from `SubmitJoin`:

- `ok`
- `id`
- `magicLinkSent`
- `signedIn`

The result UI should show storage success/failure, signed-in state, check-inbox state, and
retry/contact copy when the email handoff fails after the record is saved.

## Tests

Update Node tests for browser wiring and Go tests for handlers. Required coverage:

- Join saves a record and sends one server-side magic link for guests.
- Signed-in Join saves a record and does not send a magic link.
- Invalid Join data does not save or send email.
- Standalone `SendMagicLink` is account-enumeration-resistant.
- Disallowed origins are rejected before side effects.
- Static regression: Join completion must not call `SendMagicLink` directly.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Run `go test ./...` in `functions/firebase-go`.
