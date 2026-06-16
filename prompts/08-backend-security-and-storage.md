# Backend Security and Storage Prompt

Use this prompt before changing Netlify Functions, stored records, Identity-backed data
access, or any backend-adjacent form flow.

## Goal

Keep backend behaviour secure, testable, and consistent while the product grows from the
current Join and vehicle-basics storage into fuller evidence collection.

## Current Backend Stack

- Netlify Functions for validation, permissions, form handling, and data access.
- Netlify Blobs store named `owner-submissions`.
- Netlify Identity for authentication and roles.
- Node tests with `node --test`.

## Non-Negotiable Security Rules

- Every Function that receives or returns private data must verify Netlify Identity
  server-side through `context.clientContext.user`.
- Admin Functions must verify both authentication and the `admin` role in
  `user.app_metadata.roles`.
- Never store raw VINs in public static files, logs, test fixtures, or Blob records.
- Do not use plain SHA-256 for VIN deduplication.
- Use HMAC-SHA-256 with `VIN_PEPPER` from Netlify environment variables for VIN
  deduplication.
- If a VIN is provided and `VIN_PEPPER` is missing, reject the write. Do not store a weak
  hash or a full VIN.
- Never log raw VINs, uploaded evidence contents, full personal records, Identity tokens,
  request bodies, or Identity response bodies.
- Function logs may include request IDs, methods, origins, response status codes,
  submission IDs, short one-way email/user fingerprints, and boolean diagnostics.
- Reject disallowed browser origins before any side effects such as sending Identity emails
  or writing records.
- Validate all input server-side, even when the browser form already validates it.
- Validate uploaded evidence server-side by size, type, and content checks before storage
  when uploads are implemented.

## Storage Model

Use Netlify Blobs initially:

- Join records: `join/<submission-id>.json`
- Vehicle basics: `vehicle-basics/<identity-user-id>/<submission-id>.json`
- Submission IDs use `<prefix>_<uuid>`.
- Metadata is attached to each blob for future filtering and review.
- Store owner/member metadata separately from vehicle evidence where practical.
- Store evidence files as separate blobs with generated names when uploads are implemented.
- Store public aggregate statistics as a derived cache.

Consider Netlify Database/Postgres later if filtering, querying, exports, moderation
workflows, or aggregate generation outgrow object storage.

## Data Model Guidance

Separate:

- Identity user ID.
- Contact metadata.
- Vehicle evidence.
- VIN HMAC.
- Review status.
- Verification level.
- Evidence file metadata.
- Public aggregate inclusion status.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Add or update tests for server-side validation, authorization, storage shaping, and
  security boundaries.
- Confirm no real owner data, raw VINs, Identity tokens, or private evidence files are
  committed or generated into static output.
