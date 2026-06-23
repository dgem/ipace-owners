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
- `SendMagicLink` remains available at `POST /api/send-magic-link` for registered members
  who need a sign-in link without resubmitting Join.

## Magic-link rules

- Do not make Join completion fire a second browser request to `SendMagicLink`.
- Custom passwordless sign-in forms outside Join may call `SendMagicLink`, but this is a
  login path only, not registration.
- Do not add a password login UI.
- `SendMagicLink` must check the Join submissions collection before triggering any email
  side effect. If the email has not registered, or if the registration lookup fails, return
  generic success without calling Firebase Identity Toolkit. Do not reveal whether the email
  address is registered.
- For registered emails, `SendMagicLink` calls Firebase Identity Toolkit server-side and
  returns generic success for syntactically valid requests.
- Reject disallowed origins before triggering email side effects.
- Log email fingerprints, masked email addresses, origin/continue-host diagnostics, and
  provider status/error summaries. On success, log response size and whether the provider
  echoed the expected email without logging the raw response. Never log raw email addresses,
  request bodies, Identity tokens, action links, or full provider response bodies.
- Configure `FIREBASE_EMAIL_LINK_DOMAIN` from the environment's verified Firebase Hosting
  custom domain and pass it as Identity Toolkit's `linkDomain`. Keep
  `FIREBASE_EMAIL_CONTINUE_URL` as the post-action account URL. Firebase web API keys are
  public project identifiers and may appear in action URLs; restrict them to the required
  Firebase APIs, but do not treat their presence in an email link as credential exposure.
- For PR deployments, derive `FIREBASE_EMAIL_CONTINUE_URL` from that PR's generated Firebase
  Hosting preview URL rather than a shared staging custom domain. Omit `linkDomain` because
  Firebase rejects preview/default `web.app` domains for that field. Before exercising Auth,
  append the validated project-owned preview hostname to Firebase Auth's `authorizedDomains`
  without removing permanent entries; replace stale PR preview hostnames to keep the list
  bounded.
- If the same email address submits Join more than once, keep the browser response generic
  but log that the email hash has previous Join submissions so operators can distinguish
  repeat attempts from first-time registration.
- Treat a successful Identity Toolkit response as request acceptance, not proof of mailbox
  delivery. User-facing copy must not claim that an email was delivered or sent; advise
  checking spam/filtering and requesting another link.
- Keep Firebase email-link delivery troubleshooting aligned with
  `17-operations-ci-and-troubleshooting.md`, including sending quotas, spam/junk checks,
  sender-template settings, and the option to generate action links with the Admin SDK and
  send through a transactional email/SMTP provider when delivery tracking is required.

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
- Standalone `SendMagicLink` suppresses email side effects for unregistered addresses.
- Disallowed origins are rejected before side effects.
- Static regression: Join completion must not call `SendMagicLink` directly.

## Validation

- Run `make test`.
- Run `make build`.
