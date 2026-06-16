# Identity Magic Link and Join Functions Prompt

Use this prompt when changing:

- `netlify/functions/lib/identity-magic-link.js`
- `netlify/functions/send-magic-link.js`
- `netlify/functions/submit-join.js`
- Join form submission wiring in `src/join.njk` or `src/assets/js/identity.js`

## Goal

Maintain a single browser submission for joining the group while preserving a standalone
magic-link request path for existing users.

## Current Flow

- The Join form posts once to `/.netlify/functions/submit-join`.
- `submit-join.js` validates required Join fields and consent.
- `submit-join.js` stores the Join record in Postgres and regenerates the private
  member/account JSON snapshot.
- For logged-out users, `submit-join.js` calls shared server-side magic-link code in the
  same request.
- For signed-in users, `submit-join.js` stores the record and does not send another magic
  link.
- `send-magic-link.js` remains available as a standalone endpoint for users who already
  have an account and need another sign-in link without resubmitting Join.

## Magic Link Rules

- Do not call Netlify Identity REST APIs directly from browser code.
- Do not make Join completion fire a second browser request to `send-magic-link`.
- Keep the shared Identity flow in `identity-magic-link.js`.
- The shared flow attempts Identity `/signup` first and falls back to `/recover` for an
  existing account response.
- Valid same-origin Identity failures should be reported without revealing whether the
  email address is registered.
- `NETLIFY_IDENTITY_BASE_URL` may be used locally because Netlify Dev does not expose a
  local Identity API.
- Do not fall back to localhost as the Identity API base.

## Join Record Shape

Join records include:

- `id`, `type`, `createdAt`, `updatedAt`.
- optional `identityUserId`.
- `contact`: name, email, country.
- `membership`: one combined relationship/ownership value plus skills. Do not store a
  separate `ownership` field for new submissions.
- `consents`: contact, not-legal-claim acknowledgement, anonymised analysis.
- `review`: status and verification level.

After writing the Join record, regenerate the member/account snapshot used by
`member-data.js`. Do not write private member snapshots to public static output.

Accepted relationship values are:

- `current-owner-one`
- `current-owner-multiple`
- `former-owner`
- `prospective-buyer`
- `helping-owner`
- `trade-specialist`
- `other`

## Browser Result State

`identity.js` should read the JSON response from `submit-join`:

- `ok`
- `id`
- `magicLinkSent`
- `signedIn`

The result UI should show:

- database success/failure;
- signed-in state for already-authenticated users;
- check-inbox state when `magicLinkSent` is true for a logged-out user;
- retry/contact copy when the record saves but the Identity email handoff fails.

## Tests

Update:

- `test/submit-join.test.js`
- `test/send-magic-link.test.js`
- `test/join-single-post.test.js`

Required coverage:

- Join saves a record and sends one server-side magic link for guests.
- Signed-in Join saves a record and does not send a magic link.
- Invalid Join data does not save or send email.
- Standalone `send-magic-link` handles signup and recover flows.
- Disallowed origins are rejected before Identity side effects.
- Static regression: Join form/JS must not call `send-magic-link` directly.

## Validation

- Run `npm test`.
- Run `npm run build`.
