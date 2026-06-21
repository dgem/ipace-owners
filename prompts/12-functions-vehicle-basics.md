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
- The Go `SubmitVehicleBasics` Function verifies the ID token server-side.
- Records are stored in Firestore and the private member/account snapshot is regenerated.

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

## VIN rules

- Full VINs are never stored.
- Normalize VINs server-side before validation.
- Valid VINs use `[A-HJ-NPR-Z0-9]{17}`.
- If a VIN is provided, `VIN_PEPPER` must be configured.
- Store a VIN HMAC and final six characters only.
- If a VIN is provided and `VIN_PEPPER` is missing, do not store or derive any VIN
  identifier. If registration is present, save the registration-based record and log a
  warning. If VIN is the only identifier, reject the write with a clear configuration
  message.

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

## Tests

Update Node tests for browser wiring and Go tests for handler behavior. Required coverage:

- Unauthenticated requests return 401.
- Missing VIN and registration returns 400.
- VIN-only writes require `VIN_PEPPER`; VIN plus registration can save without a VIN
  identifier when the secret is missing.
- Stored records never include the full VIN.
- Vehicle and battery fields are cleaned and shaped correctly.
- Firestore-backed saves trigger private member snapshot regeneration.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Run `go test ./...` in `functions/firebase-go`.
