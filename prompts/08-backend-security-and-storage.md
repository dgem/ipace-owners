# Backend Security and Storage Prompt

Use this prompt before changing Netlify Functions, stored records, Identity-backed data
access, or any backend-adjacent form flow.

## Goal

Keep backend behaviour secure, testable, and consistent while the product grows from the
current Join and vehicle-basics storage into fuller evidence collection.

## Current Backend Stack

- Netlify Functions for validation, permissions, form handling, and data access.
- Netlify Database/Postgres for structured owner data.
- Netlify Blobs only for binary evidence uploads.
- Netlify Identity for authentication and roles.
- Node tests with `node --test`.

## Non-Negotiable Security Rules

- Every Function that receives or returns private data must verify Netlify Identity
  server-side through `context.clientContext.user`.
- Admin Functions must verify both authentication and the `admin` role in
  `user.app_metadata.roles`.
- Never store raw VINs in public static files, logs, test fixtures, Blob records, or
  Postgres rows.
- Do not use plain SHA-256 for VIN deduplication.
- Use HMAC-SHA-256 with `VIN_PEPPER` from Netlify environment variables for VIN
  deduplication.
- If a VIN is provided and `VIN_PEPPER` is missing, do not store or derive any VIN
  identifier. If registration is present, save the registration-based record; if VIN is the
  only identifier, reject the write with a clear configuration message.
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

Use Netlify Database/Postgres for structured data:

- Members, Join submissions, vehicles, battery readings, review state, and audit events are
  stored in relational tables.
- A member can have multiple vehicle records.
- Member/account JSON snapshots are generated after signup and vehicle changes, stored
  privately, and served only after server-side Identity verification.
- Public aggregate JSON snapshots are generated from reviewed/anonymised data for static
  dashboard rendering.
- Store evidence files as separate blobs with generated names when uploads are implemented;
  keep ownership, permissions, and review metadata in Postgres.

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
