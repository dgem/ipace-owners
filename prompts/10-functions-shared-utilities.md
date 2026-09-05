# Shared Function Utilities Prompt

Use this prompt when changing shared Go helpers in `functions/firebase-go`.

## Goal

Keep shared Function behaviour small, explicit, privacy-safe, and covered by tests.

## Current utilities

Shared Go helpers own:

- CORS headers and browser origin allowlist.
- The single `Api` HTTP router that dispatches Firebase Hosting `/api/**` requests to the
  in-process handler functions.
- Structured privacy-safe logging.
- JSON body parsing.
- Conservative input cleaning and enum/date/number validation helpers.
- Email fingerprinting.
- VIN HMAC generation.
- Firebase ID-token verification.
- Opaque support-code propagation and trace-scoped authorization-decision logging.
- Firestore and Cloud Storage client setup.
- Private member/account JSON snapshot helpers.
- Submission ID generation.

## Rules

- Keep helpers dependency-light and idiomatic Go.
- Do not add business-specific validation unless it is genuinely shared.
- Do not log raw personal data, VINs, tokens, request bodies, or provider responses.
- `requireUser` and `requireAdmin` are the authorization decision point. When a request has a
  valid opaque `X-Ipace-Auth-Trace`, log exactly the route, required role, decision and status
  for every allow or deny decision. Do not create uncorrelated authorization logs for requests
  without a support code.
- Preserve a valid support code through a passwordless email-link `continueUrl` using the
  `authTrace` query key only; reject invalid values and never use it as authorization data.
- `AuthDiagnostics` is browser-only telemetry: require a non-empty allowed `Origin` and a valid
  `X-Ipace-Auth-Trace` that exactly matches its bounded JSON body before recording an event.
- Keep origin checks reusable and call them before side effects in every public Function.
  Allowed Firebase preview origins must be scoped to the current Firebase/GCP project
  prefix, not every `web.app` or `firebaseapp.com` host.
- Derive Firebase email-link `continueUrl` from a validated request origin for allowed
  preview/local/production origins, and fall back to the configured environment account URL
  for non-browser or disallowed origins.
- Preserve repeated form field parsing for checkbox groups at the browser/API boundary.
- Keep storage access behind shared helpers so handlers can be tested without live GCP.
- Keep generated JSON snapshot helpers separate from validation helpers.

## Tests

Update Go tests when utility behaviour changes. Coverage should include API routing,
origins, Firebase preview-origin scoping, dynamic email-link continue URL derivation, JSON
body parsing, email/date/enum/integer/decimal cleaning, HMAC/fingerprint behaviour, and auth
claim handling. Cover trace validation, continue-URL propagation, and member/admin allow and
deny decision paths.

## Validation

- Run `GOCACHE=/tmp/ipace-owners-go-build make test-go` or `go test ./...` in `functions/firebase-go`.
- Run `make test`.
- Run `make build`.
