# Backend Roadmap, Functions, and Blobs Prompt

Use this prompt before implementing real submission persistence or admin data access.

## Goal

Evolve the static MVP into a data-collecting product using Netlify Functions and server-side storage while protecting personal data and vehicle identifiers.

## Non-negotiable security rules

- Every Function that receives or returns private data must verify the Netlify Identity JWT server-side.
- Admin Functions must verify both authentication and the `admin` role in `app_metadata.roles`.
- Never store raw VINs in public static files.
- Do not use plain SHA-256 for VIN deduplication.
- Use HMAC-SHA-256 with a secret pepper from Netlify environment variables for VIN deduplication.
- Never log raw VINs, uploaded evidence contents, full personal records, or Identity tokens.
- Function logs may include structured diagnostics such as request IDs, methods, origins,
  response status codes, and short one-way email fingerprints for troubleshooting. Do not log
  raw email addresses, names, VINs, Identity tokens, request bodies, or Identity response bodies.
- Validate all input server-side.
- Validate uploaded evidence server-side by size, type, and content checks.

## Implemented Functions

- `send-magic-link.js`: accepts `POST { email, name }` from the Join form, calls Netlify
  Identity `/signup` (new users) or `/recover` (existing users) server-side, and returns
  account-enumeration-resistant responses for valid same-origin requests. It may return
  non-200 responses for disallowed origins, invalid JSON, invalid email input, or unsupported
  methods. CORS and explicit Origin checks restrict browser calls to same-site origins. It
  logs privacy-safe structured diagnostics to help debug Identity email delivery issues.
- In local Netlify Dev, `send-magic-link.js` must use `NETLIFY_IDENTITY_BASE_URL` to point
  at the deployed site's `/.netlify/identity` endpoint. Netlify Dev does not expose a local
  Identity API, so localhost must not be used as an Identity base URL.

## Implemented storage

- The Join form is a Netlify Form named `join`. It stores membership expressions of
  interest, including contact details, relationship/ownership answers, volunteering
  interests, and consent choices.
- JavaScript-enhanced Join submission posts URL-encoded `FormData` to the same Netlify Form
  while keeping the multi-step completion screen visible. No-JavaScript users can still
  submit the normal HTML form.
- Treat Netlify Forms as the interim membership-intake store. Do not add a duplicate
  `submit-join.js` Function unless the product needs stronger validation, Identity-linked
  profiles, moderation workflows, or migration into Blobs/Database.

## Proposed Functions

- `submit-join.js`: optional future replacement for Netlify Forms if membership intake needs authenticated profiles, richer validation, or migration into Blobs/Database.
- `submit-vehicle.js`: authenticated vehicle evidence submission.
- `get-member-data.js`: returns only the current member's submissions.
- `admin-review.js`: returns pending submissions to admins only.
- `admin-update-submission.js`: changes review status and verification level.
- `public-stats.js`: returns anonymised aggregate statistics for public dashboard rendering or static regeneration.

## Storage model

Use Netlify Blobs initially:

- Store submission records as JSON blobs keyed by generated submission IDs.
- Store owner/member metadata separately from vehicle evidence where practical.
- Store evidence files as separate blobs with generated names.
- Store public aggregate statistics as a derived cache.

Consider Netlify Database/Postgres later if filtering, querying, exports, or moderation workflows outgrow object storage.

## Data model guidance

Separate:

- Identity user ID.
- Contact metadata.
- Vehicle evidence.
- VIN HMAC.
- Review status.
- Verification level.
- Evidence file metadata.
- Public aggregate inclusion status.

## UI integration

- The Join form already calls `/.netlify/functions/send-magic-link` to dispatch a
  Netlify Identity sign-in email. Do not revert this to a direct Identity API call.
- The Join form also submits to Netlify Forms. Keep the Netlify Forms save and
  `send-magic-link` result handling independent so one failure does not mask the other.
- `send-magic-link` must reject disallowed origins before making any Identity API calls so
  cross-site `no-cors` requests cannot be used to trigger unsolicited emails.
- The browser should interpret the JSON response body's `ok` flag, not `response.ok` alone,
  because same-origin Identity errors intentionally return HTTP 200 with `{ ok: false }`.
- Replace placeholder vehicle/evidence form submission handling with `fetch` calls
  to the relevant Functions only after those Functions exist.
- Send Identity JWTs in the `Authorization` header where authentication is required.
- Preserve accessible validation and progressive enhancement.
- Show clear success, retryable failure, and non-retryable validation error states.

## Validation

- Add tests for server-side validation and authorization.
- Test unauthenticated, authenticated non-admin, and admin paths.
- Run `npm run build`.
- Do not deploy broader vehicle/evidence data collection until the privacy policy has been formally reviewed.
