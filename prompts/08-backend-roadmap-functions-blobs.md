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
- Validate all input server-side.
- Validate uploaded evidence server-side by size, type, and content checks.

## Implemented Functions

- `send-magic-link.js`: accepts `POST { email, name }` from the Join form, calls Netlify
  Identity `/signup` (new users) or `/recover` (existing users) server-side, and returns
  account-enumeration-resistant responses for valid same-origin requests. It may return
  non-200 responses for disallowed origins, invalid JSON, invalid email input, or unsupported
  methods. CORS and explicit Origin checks restrict browser calls to same-site origins.

## Proposed Functions

- `submit-join.js`: authenticated or guest membership expression of interest, depending on registration policy.
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
- Do not deploy live data collection until the privacy policy has been formally reviewed.
