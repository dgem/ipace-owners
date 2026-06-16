# Future Evidence and Statistics Functions Prompt

Use this prompt before implementing evidence sections beyond vehicle basics, evidence
uploads, admin review mutation, exports, or public aggregate statistics.

## Goal

Extend the stored evidence model without weakening privacy, authorization, or data quality.

## Future Functions

Potential future Functions:

- `submit-vehicle.js`: authenticated full vehicle evidence submission for recall, repair,
  support, loan car, payment, responsibility, consent-review, and evidence-upload metadata.
- `admin-update-submission.js`: admin-only review status, verification level, exclusion,
  and moderation updates.
- `publish-public-stats.js`: admin-only generation of anonymised aggregate statistics for
  public static JSON snapshots.
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
- Public pages should read generated static JSON snapshots rather than querying Firestore on
  every page view.
- Do not publish full VINs, registrations, owner names, email addresses, uploaded
  documents, or individual case narratives without explicit permission.
- Support verification levels:
  - self-reported;
  - document supplied;
  - reviewed by organiser;
  - duplicate checked;
  - excluded from public statistics.
- Aggregates must exclude records marked as excluded from public statistics.
- Public static JSON can be written to locations such as
  `/assets/data/public-evidence-summary.json`. Private member/account JSON must not be
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

- Run `npm test`.
- Run `npm run build`.
- Manually inspect public output for accidental private data exposure.
