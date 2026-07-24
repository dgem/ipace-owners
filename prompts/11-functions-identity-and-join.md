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
- For registered emails, `SendMagicLink` uses the configured server-side delivery path and
  returns generic success for syntactically valid requests: branded Admin-SDK/Resend delivery
  when configured, otherwise Firebase Identity Toolkit's default sender.
- Reject disallowed origins before triggering email side effects.
- Log email fingerprints, masked email addresses, origin/continue-host diagnostics, and
  provider status/error summaries. On success, log response size and whether the provider
  echoed the expected email without logging the raw response. Never log raw email addresses,
  request bodies, Firebase ID tokens, action links, or full provider response bodies.
- Configure `FIREBASE_EMAIL_LINK_DOMAIN` from the environment's verified Firebase Hosting
  custom domain and pass it as Identity Toolkit's `linkDomain`. If the environment variable
  is absent, derive `linkDomain` from the custom-domain `continueUrl` so production links
  use `ipace-owners.org` rather than the default `firebaseapp.com` action host. Keep
  `FIREBASE_EMAIL_CONTINUE_URL` as the post-action account URL. Firebase web API keys are
  public project identifiers and may appear in action URLs; restrict them to the required
  Firebase APIs, but do not treat their presence in an email link as credential exposure.
- If `RESEND_API_KEY` and `RESEND_FROM` are configured, use Firebase Admin
  `EmailSignInLink` to generate the passwordless link without asking Firebase to send the
  email, then send a branded HTML and plain-text email through Resend. Include the launch
  hero image from `/images/ipace-hero.png` using an absolute asset URL. For PR preview
  continue URLs, use the PR preview origin for email image assets so the email points at the
  image deployed with that branch. Keep Firebase's default email delivery as the automatic
  fallback when Resend is not configured or fails.
  Never log the generated action link, API key, or raw provider response body.
- For PR deployments, derive `FIREBASE_EMAIL_CONTINUE_URL` from that PR's generated Firebase
  Hosting preview request origin rather than a shared staging custom domain or a value baked
  into Function environment variables. Omit `linkDomain` because Firebase rejects
  preview/default `web.app` domains for that field. Before exercising Auth, append the
  validated project-owned preview hostname to Firebase Auth's `authorizedDomains` without
  removing permanent entries; replace stale PR preview hostnames to keep the list bounded.
  Do not accept arbitrary `web.app` origins; the host must match the current Firebase
  project preview pattern.
- If the same email address submits Join more than once, keep the browser response generic
  but log that the email hash has previous Join submissions so operators can distinguish
  repeat attempts from first-time registration.
- Treat a successful Identity Toolkit response as request acceptance, not proof of mailbox
  delivery. User-facing copy must not claim that an email was delivered or sent; advise
  checking spam/filtering and requesting another link.
- The browser must remember the Join email in `ipaceEmailForSignIn` after a successful
  guest Join response with `magicLinkSent: true`, so the first clicked registration link can
  complete without an extra prompt in the same browser.
- If the remembered email is unavailable or wrong, complete the pending Firebase email link
  through the visible sign-in form. Do not use blocking browser prompts for this flow.
- Keep Firebase email-link delivery troubleshooting aligned with
  `17-operations-ci-and-troubleshooting.md`, including sending quotas, spam/junk checks,
  sender-template settings, and the implemented Admin SDK plus Resend delivery option.
- The versioned account-action HTML templates are retained as future assets but are not
  applied while the product uses passwordless email-link sign-in. Firebase's built-in
  `EMAIL_SIGNIN` body is fixed. Branded sign-in copy is provided only by the implemented
  server-generated Admin SDK link plus Resend path; when Firebase performs default delivery,
  do not claim its subject, body, reply-to address, or sender name are template-managed.

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
- Provide an operator-only Join re-engagement command for staging and production. It must extract
  Join submissions and Firebase Auth users itself, then suppress accounts matched by exact email,
  email canonicalised without `+tag` addressing, or normalised display name. It must default to dry
  run, write a permission-restricted audit CSV and secret-free settings manifest, generate a stable
  environment/date campaign ID when one is not supplied, require an exact eligible count and typed
  interactive confirmation for a live send, refresh the complete comparison before confirmation,
  generate a fresh Firebase sign-in link per recipient, pace Resend calls, use idempotency keys,
  and refuse to overwrite results or manifests. Read the same `RESEND_*` environment names as the
  deployed app. Info logs expose only counts; debug logs may
  print candidate names and addresses, but action links must never be logged. Do not expose this
  workflow as a public endpoint.
- Treat a canonical Firebase Auth email without a canonical Join email as a privacy-invariant
  failure. Report it in dry-run output, return failure after writing the audit ledger, and always
  block live sending. Name matching may suppress a reminder for safety but must not satisfy this
  strict Auth-to-Join coverage check.
- The browser-managed re-engagement email should address the member by first name, state when they
  submitted Join, lead with a secure per-recipient “Verify my account details” CTA, explain the
  member access available after verification, state the link lifetime, and provide the contact
  address for recipients who did not ask to join or have changed their mind. Keep its Markdown
  prose routinely editable. Regression tests should protect personalisation, date rendering,
  escaping, the unique action link, shared HTML chrome/hero/CTA, the mandatory consent footer and
  complete template-field substitution without pinning editorial sentences.
- Invalid Join data does not save or send email.
- Standalone `SendMagicLink` is account-enumeration-resistant.
- Standalone `SendMagicLink` suppresses email side effects for unregistered addresses.
- Disallowed origins are rejected before side effects.
- Static regression: Join completion must not call `SendMagicLink` directly.

## Validation

- Run `make test`.
- Run `make build`.
