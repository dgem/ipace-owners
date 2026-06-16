# Backend Security and Storage Prompt

Use this prompt before changing Go Cloud Functions, stored records, Firebase-backed data
access, or any backend-adjacent form flow.

## Goal

Keep backend behaviour secure, testable, and consistent while the product grows from the
current Join and vehicle-basics storage into fuller evidence collection.

## Current backend stack

- Go Cloud Functions for validation, permissions, form handling, and data access.
- Firebase Authentication for member sessions and admin custom claims.
- Cloud Firestore for structured owner data.
- Cloud Storage for generated JSON snapshots and future binary evidence uploads.
- OpenTofu/Terraform for GCP resource configuration.

## Non-negotiable security rules

- Every Function that receives or returns private data must verify Firebase ID tokens
  server-side.
- Admin Functions must verify both authentication and the `admin` role/custom claim.
- Never store raw VINs in public static files, logs, test fixtures, Firestore, or Cloud
  Storage.
- Do not use plain SHA-256 for VIN deduplication.
- Use HMAC-SHA-256 with `VIN_PEPPER` from GCP Secret Manager / Function env vars.
- If a VIN is provided and `VIN_PEPPER` is missing, do not store or derive any VIN
  identifier. If registration is present, save the registration-based record; if VIN is the
  only identifier, reject the write with a clear configuration message.
- Never log raw VINs, uploaded evidence contents, full personal records, Identity tokens,
  request bodies, or provider response bodies.
- Function logs may include request IDs, methods, origins, response status codes,
  submission IDs, short one-way email/user fingerprints, and boolean diagnostics.
- Reject disallowed browser origins before side effects such as sending emails or writing
  records.
- Validate all input server-side, even when the browser form already validates it.
- Validate uploaded evidence server-side by size, type, and content checks before storage.

## Storage model

Use Firestore for structured data:

- Members, Join submissions, vehicles, battery readings, review state, and audit events are
  stored as documents.
- A member can have multiple vehicle records.
- Member/account JSON snapshots are generated after signup and vehicle changes, stored
  privately, and served only after server-side Firebase verification.
- Public aggregate JSON snapshots are generated from reviewed/anonymised data for static
  dashboard rendering.
- Store evidence files as Cloud Storage objects with generated names; keep ownership,
  permissions, and review metadata in Firestore.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Run `go test ./...` in `functions/firebase-go`.
- Add or update tests for server-side validation, authorization, storage shaping, and
  security boundaries.
- Confirm no real owner data, raw VINs, Identity tokens, or private evidence files are
  committed or generated into static output.
