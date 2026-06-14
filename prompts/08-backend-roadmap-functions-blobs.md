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

- `send-magic-link.js`: standalone endpoint for requesting another sign-in link without
  resubmitting the Join form. It accepts `POST { email, name }`, calls shared server-side
  magic-link code, and returns account-enumeration-resistant responses for valid same-origin
  requests. It may return non-200 responses for disallowed origins, invalid request bodies,
  invalid email input, or unsupported methods. CORS and explicit Origin checks restrict
  browser calls to same-site origins. It logs privacy-safe structured diagnostics to help
  debug Identity email delivery issues.
- In local Netlify Dev, `send-magic-link.js` must use `NETLIFY_IDENTITY_BASE_URL` to point
  at the deployed site's `/.netlify/identity` endpoint. Netlify Dev does not expose a local
  Identity API, so localhost must not be used as an Identity base URL.
- `submit-join.js`: accepts Join form submissions, validates required membership consent,
  stores membership interest in Netlify Blobs, and sends the Identity magic link in the
  same request for logged-out users.
- `submit-vehicle-basics.js`: accepts signed-in vehicle basics submissions, validates the
  initial vehicle and battery health slice, stores records in Netlify Blobs, and stores VIN
  HMACs rather than full VINs.

## Implemented storage

- Join records are stored under `join/<submission-id>.json`.
- Vehicle basics records are stored under
  `vehicle-basics/<identity-user-id>/<submission-id>.json`.
- `VIN_PEPPER` must be configured before collecting VINs. If a VIN is provided and
  `VIN_PEPPER` is missing, `submit-vehicle-basics.js` must reject the write rather than
  store a weak hash or full VIN.

## Proposed Functions

- `submit-vehicle.js`: authenticated full vehicle evidence submission for recall, repair,
  support, payment, responsibility, consent-review, and evidence-upload metadata.
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

- The Join form must make exactly one browser request to `/.netlify/functions/submit-join`.
  `submit-join` stores the record and calls shared server-side magic-link code for logged-out
  users.
- Keep `/.netlify/functions/send-magic-link` for standalone sign-in link requests, but do
  not call it separately from Join completion.
- Signed-in vehicle basics submit to `/.netlify/functions/submit-vehicle-basics` with an
  Identity JWT in the `Authorization` header.
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
- Run `npm test`.
- Run `npm run build`.
- Do not deploy broader vehicle/evidence data collection until the privacy policy has been formally reviewed.
