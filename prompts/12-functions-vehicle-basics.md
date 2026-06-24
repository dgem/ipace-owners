# Vehicle Basics Function Prompt

Use this prompt when changing:

- `functions/firebase-go` vehicle basics handler
- `src/submit-vehicle-data.njk`
- vehicle-basics submission behavior in `src/assets/js/identity.js`

## Goal

Maintain the first database-backed vehicle evidence slice: signed-in members can register
basic vehicle identity, ownership, mileage, and battery State of Health details without
storing full VINs. Members may register multiple vehicles.

## Current flow

- The vehicle data page is a two-step vehicle-basics form.
- It requires a signed-in Firebase Auth user.
- The browser posts JSON to `POST /api/submit-vehicle-basics`.
- `identity.js` sends a Firebase ID token in the `Authorization` header for forms marked
  with `data-database-requires-auth`.
- The form must validate that at least one identifier, VIN or registration, is present
  before the user leaves the vehicle-details step. Use generic multistep validation
  attributes rather than custom one-off page JavaScript.
- Client-side validation should hard-block future dates and VIN-only invalid VINs. It
  should soft-warn, not block, when a GB registration does not look like a common UK
  registration format or when a syntactically valid VIN does not look like a typical
  Jaguar/JLR VIN.
- The Go `SubmitVehicleBasics` Function verifies the ID token server-side.
- Records are stored in Firestore and the private member/account snapshot is regenerated.
- An initial SoH value creates an append-only `batteryReadings` record.
- Members append later readings with `POST /api/submit-soh`; `SubmitSOH` verifies that the
  authenticated UID owns the referenced vehicle before writing.
- Vehicle and SoH writes regenerate both the private member snapshot and consent-filtered
  public aggregate snapshot.

## Collected fields

Vehicle details:

- VIN or registration; at least one is required.
- Country.
- Model year.
- Current mileage.
- Owned since.
- First registration date.

Battery health:

- Current State of Health.
- Date SoH was measured.
- Mileage at SoH measurement.
- Source of SoH reading.

Date fields must not be in the future. Enforce this both in the browser and in
`SubmitVehicleBasics` before writing Firestore records.

## VIN rules

- Full VINs are never stored.
- Normalize VINs server-side before validation.
- Valid VINs use `[A-HJ-NPR-Z0-9]{17}`.
- Treat VIN as optional when registration is provided. If a member supplies registration and
  a malformed VIN, save a registration-based record, ignore the malformed VIN, and log a
  sanitized warning. Reject malformed VINs only when VIN is the sole vehicle identifier.
- If a VIN is provided, `VIN_PEPPER` must be configured.
- Store a VIN HMAC and final six characters only.
- If a VIN is provided and `VIN_PEPPER` is missing, do not store or derive any VIN
  identifier. If registration is present, save the registration-based record and log a
  warning. If VIN is the only identifier, reject the write with a clear configuration
  message.

## Frontend copy and result state

- The vehicle page callout should say that this form registers one I-PACE and that, after
  saving, members can use My Data to add further SoH readings and service/fault history.
- Do not describe service or fault history as unavailable now that the member workspace
  supports it.
- Keep fuller recall, repair, loan car, payment, goodwill and evidence upload collection
  framed as future evidence-workflow expansion until those fields are implemented.
- The result panel must reflect the actual API outcome: success icon and success actions
  only after the API returns `ok: true`; auth and save failures must show a non-success icon,
  a clear message, and no "add another vehicle" success action.
- When the API returns a validation or configuration error, display the server-provided
  message in the result panel so testers can diagnose failures without opening DevTools.

## Record shape

Vehicle-basics records include:

- `id`, `type`, `createdAt`, `updatedAt`.
- `identityUserId`.
- `userEmailHash`.
- `vehicle`: `vinHash`, `vinLast6`, registration, country, modelYear, mileage,
  ownedSince, firstRegistrationDate.
- `battery`: stateOfHealth, measuredAt, mileageAtMeasurement, source.
- `review`: status and verification level.

Keep the API response shape compatible with `member-auth.js` unless changing that client
code deliberately.

SoH reading records include `id`, timestamps, `identityUserId`, `vehicleId`, the battery
measurement fields, and review metadata. Earlier measurements must never be overwritten.

## Tests

Update Node tests for browser wiring and Go tests for handler behavior. Required coverage:

- Unauthenticated requests return 401.
- Missing VIN and registration returns 400.
- VIN-only writes require `VIN_PEPPER`; VIN plus registration can save without a VIN
  identifier when the secret is missing.
- Stored records never include the full VIN.
- Vehicle and battery fields are cleaned and shaped correctly.
- Firestore-backed saves trigger private member snapshot regeneration.
- SoH updates reject unauthenticated users, non-owners, missing dates/sources, and values
  outside 0-100.
- Public aggregates use the latest SoH per car for fleet averages and first-to-latest values
  for degradation statistics.

## Validation

- Run `make test`.
- Run `make build`.
- Run `GOCACHE=/tmp/ipace-owners-go-build make test-go` or `go test ./...` in `functions/firebase-go`.
