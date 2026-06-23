# Shared Function Utilities Prompt

Use this prompt when changing shared Go helpers in `functions/firebase-go`.

## Goal

Keep shared Function behaviour small, explicit, privacy-safe, and covered by tests.

## Current utilities

Shared Go helpers own:

- CORS headers and browser origin allowlist.
- Structured privacy-safe logging.
- JSON body parsing.
- Conservative input cleaning and enum/date/number validation helpers.
- Email fingerprinting.
- VIN HMAC generation.
- Firebase ID-token verification.
- Firestore and Cloud Storage client setup.
- Private member/account JSON snapshot helpers.
- Submission ID generation.

## Rules

- Keep helpers dependency-light and idiomatic Go.
- Do not add business-specific validation unless it is genuinely shared.
- Do not log raw personal data, VINs, tokens, request bodies, or provider responses.
- Keep origin checks reusable and call them before side effects in every public Function.
- Preserve repeated form field parsing for checkbox groups at the browser/API boundary.
- Keep storage access behind shared helpers so handlers can be tested without live GCP.
- Keep generated JSON snapshot helpers separate from validation helpers.

## Tests

Update Go tests when utility behaviour changes. Coverage should include origins, JSON body
parsing, email/date/enum/integer/decimal cleaning, HMAC/fingerprint behaviour, and auth
claim handling.

## Validation

- Run `GOCACHE=/tmp/ipace-owners-go-build make test-go` or `go test ./...` in `functions/firebase-go`.
- Run `make test`.
- Run `make build`.
