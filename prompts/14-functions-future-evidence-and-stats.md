# Future Evidence and Statistics Functions Prompt

Use this prompt before implementing evidence sections beyond vehicle basics, evidence
uploads, admin review mutation, exports, or expanded public aggregate statistics.

## Goal

Extend the stored evidence model without weakening privacy, authorization, or data quality.

## Future Functions

Current public statistics are already served by the Go `PublicStats` Function from a
generated, consent-filtered aggregate snapshot. Future work should extend that path rather
than introducing a parallel browser query or static sample-data model.

Potential future Go Functions or handler extensions:

- `SubmitVehicleEvidence`: authenticated full vehicle evidence submission for recall, repair,
  support, loan car, payment, responsibility, consent-review, and evidence-upload metadata.
- `AdminUpdateSubmission`: admin-only review status, verification level, exclusion,
  and moderation updates.
- `PublishPublicStats` or an admin-only `PublicStats` publish mode: explicit regeneration
  of anonymised aggregate statistics after review changes.
- Evidence upload Functions: authenticated upload URL generation, file metadata creation,
  server-side validation, and admin review workflows.
- Export Functions: admin-only CSV/JSON exports with explicit audit logging.

## Evidence Upload Rules

- Do not implement browser-direct public uploads without server-side validation.
- Validate file size, content type, extension, and expected content before storage.
- Generate storage keys server-side.
- Store uploaded evidence separately from JSON records.
- Tell owners to redact personal information they do not want stored.
- Never publish uploaded documents without explicit permission.

## Public Statistics Rules

- Public pages may show only anonymised aggregate data.
- Public pages should read aggregate data through `GET /api/public-stats`, backed by a
  generated snapshot, rather than querying canonical Firestore records from the browser or
  rendering placeholder sample figures as live data.
- Do not publish full VINs, registrations, owner names, email addresses, uploaded
  documents, or individual case narratives without explicit permission.
- Support verification levels:
  - self-reported;
  - document supplied;
  - reviewed by organiser;
  - duplicate checked;
  - excluded from public statistics.
- Aggregates must exclude records marked as excluded from public statistics.
- Public aggregate JSON can be stored in Cloud Storage or another generated snapshot
  location and served through `PublicStats`. Private member/account JSON must not be
  written to public static output.

## Admin Mutation Rules

- Every mutation Function must require server-side admin role verification.
- Validate mutation payloads against explicit enums.
- Preserve enough metadata for auditability: who changed what and when.
- Avoid destructive deletes as the default moderation action; prefer exclusion/status fields.

## Tests

Add tests before implementing each Function:

- unauthenticated rejection;
- authenticated non-admin rejection where relevant;
- admin success path where relevant;
- validation failures;
- storage shape and metadata;
- aggregate exclusion rules;
- no raw VIN/personal-data leakage in public responses.

## Validation

- Run `make test`.
- Run `make build`.
- Manually inspect public output for accidental private data exposure.
