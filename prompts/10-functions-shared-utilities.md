# Shared Function Utilities Prompt

Use this prompt when changing `netlify/functions/lib/submission-utils.js` or any common
Function helper.

## Goal

Keep shared Function behaviour small, explicit, privacy-safe, and covered by tests.

## Current Utilities

`submission-utils.js` owns:

- CORS headers and the browser origin allowlist.
- Structured privacy-safe logging.
- Request metadata extraction.
- JSON and `application/x-www-form-urlencoded` body parsing.
- Conservative input cleaning and enum/date/number validation helpers.
- Email fingerprinting.
- HMAC generation.
- Netlify Identity user extraction from `context.clientContext.user`.
- Postgres/database access for structured owner data.
- Netlify Blobs access only for future binary evidence uploads.
- Private member/account JSON snapshot generation helpers.
- Submission ID generation.

## Rules

- Keep helpers dependency-light and CommonJS-compatible for Netlify Functions.
- Do not add business-specific validation here unless it is genuinely shared by multiple
  Functions.
- Do not log raw personal data, VINs, tokens, request bodies, or Identity API responses.
- Keep origin checks reusable and call them before side effects in every public Function.
- Preserve repeated form field parsing for checkbox groups.
- Keep storage access behind shared helpers so Functions can be tested without a live
  database.
- Keep generated JSON snapshot helpers separate from validation helpers.

## Tests

Update `test/submission-utils.test.js` when utility behaviour changes.

Required coverage:

- Allowed and disallowed origins.
- JSON body parsing.
- URL-encoded body parsing with repeated fields.
- Email/date/enum/integer/decimal cleaning.
- HMAC or fingerprint behaviour when changed.

## Validation

- Run `npm test`.
- Run `npm run build`.
