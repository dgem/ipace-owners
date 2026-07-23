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
- Never log raw VINs, uploaded evidence contents, full personal records, Firebase ID tokens,
  request bodies, or provider response bodies.
- Function logs may include request IDs, methods, origins, response status codes,
  submission IDs, short one-way email/user fingerprints, masked email addresses, provider
  error summaries, continue URL hosts, and boolean diagnostics.
- Reject disallowed browser origins before side effects such as sending emails or writing
  records.
- Treat social publishing as an external side effect: require an admin claim, validate and
  preview the exact media/caption first, reject arbitrary external media URLs, keep provider
  tokens server-side, and require an exact server-derived confirmation before calling Meta.
- Use the Function runtime identity, not an API key, for Vertex AI video generation. Grant it
  `roles/aiplatform.user` plus object access only on the private campaign-media bucket.
- Validate all input server-side, even when the browser form already validates it.
- Validate uploaded evidence server-side by size, type, and content checks before storage.

## Storage model

Use Firestore for structured data:

- Members, Join submissions, vehicles, battery readings, service events, and their review
  state are stored as documents. Audit-event storage remains future work.
- A member can have multiple vehicle records.
- Member/account JSON snapshots are generated after signed-in Join, vehicle, SoH, and
  service-event changes, stored privately, and served only after server-side Firebase
  verification. `MemberData` regenerates a missing snapshot from Firestore.
- Public aggregate JSON snapshots are regenerated after relevant writes, filtered by
  anonymised-analysis consent and review exclusion state, and served through a cacheable
  public Function for dashboard rendering without exposing canonical records.
- Store evidence files as Cloud Storage objects with generated names; keep ownership,
  permissions, and review metadata in Firestore.
- Store Veo source clips, continuation clips, extracted frames, and rejected candidates under
  the private campaign-media bucket's `work/` prefix with bounded retention. Store approved,
  auditable masters under `masters/`; retain and version them independently of working files.
- Store idempotent asynchronous generation state in `instagramGenerationJobs`. Keep provider
  operation names and GCS object names server-side. Claim each billable phase transactionally and
  fail closed on indeterminate provider/ledger transitions. Persist a safe, allowlisted failure
  classification plus provider numeric code/status for admin diagnosis; do not return arbitrary
  raw provider messages to the browser. Expose completed private masters only through short-lived,
  unguessable delivery paths whose token is stored as a hash and checked in constant time; support
  HTTP byte ranges for browser and Meta video fetches.

## Validation

- Run `make test`.
- Run `make build`.
- Add or update tests for server-side validation, authorization, storage shaping, and
  security boundaries.
- Confirm no real owner data, raw VINs, Firebase ID tokens, or private evidence files are
  committed or generated into static output.
