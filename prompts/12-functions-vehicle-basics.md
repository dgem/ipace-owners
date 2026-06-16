# Vehicle Basics Function Prompt

Use this prompt when changing:

- `netlify/functions/submit-vehicle-basics.js`
- `src/submit-vehicle-data.njk`
- vehicle-basics submission behavior in `src/assets/js/identity.js`

## Goal

Maintain the first database-backed vehicle evidence slice: signed-in members can register
basic vehicle identity, ownership, mileage, and battery State of Health details without
storing full VINs.

## Current Flow

- The vehicle data page is currently a two-step vehicle-basics form.
- It requires a signed-in Netlify Identity user.
- The browser posts JSON to `/.netlify/functions/submit-vehicle-basics`.
- `identity.js` sends the Identity JWT in the `Authorization` header for forms marked with
  `data-database-requires-auth`.
- Netlify runtime exposes the verified user as `context.clientContext.user`.
- Records are stored under
  `vehicle-basics/<identity-user-id>/<submission-id>.json`.

## Collected Fields

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

## VIN Rules

- Full VINs are never stored.
- Normalize VINs server-side before validation.
- Valid VINs use `[A-HJ-NPR-Z0-9]{17}`.
- If a VIN is provided, `VIN_PEPPER` must be configured.
- Store a VIN HMAC and final six characters only.
- Reject writes if a VIN is provided and `VIN_PEPPER` is missing.

## Record Shape

Vehicle-basics records include:

- `id`, `type`, `createdAt`, `updatedAt`.
- `identityUserId`.
- `userEmailHash`.
- `vehicle`: `vinHash`, `vinLast6`, registration, country, modelYear, mileage,
  ownedSince, firstRegistrationDate.
- `battery`: stateOfHealth, measuredAt, mileageAtMeasurement, source.
- `review`: status and verification level.

## Tests

Update `test/submit-vehicle-basics.test.js`.

Required coverage:

- Unauthenticated requests return 401.
- Missing VIN and registration returns 400.
- VIN writes require `VIN_PEPPER`.
- Stored records never include the full VIN.
- Vehicle and battery fields are cleaned and shaped correctly.

## Validation

- Run `npm test`.
- Run `npm run build`.
