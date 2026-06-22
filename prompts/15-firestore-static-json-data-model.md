# Firestore and Static JSON Data Model Prompt

Use this prompt when changing canonical structured storage or generated JSON snapshots for
member/account pages, admin review data, or public evidence statistics.

## Goal

Use Cloud Firestore as the canonical source for structured data. Generate JSON snapshots
after writes so member pages and public dashboards avoid repeated canonical-store reads.

## Canonical Data Store

Firestore is the source of truth for:

- members and Firebase Auth UID linkage;
- Join submissions, relationship status, skills, and consents;
- vehicles, with one member able to own or submit data for multiple vehicles;
- battery State of Health readings;
- future recall, repair, loan car, payment, goodwill, responsibility, and review data;
- admin review state, verification levels, and audit events.

Use Go Cloud Functions with the Firebase Admin SDK and Firestore client. Do not make
private Firestore collections directly readable from browser code.

Provision one named Firestore database per environment with a database ID matching the GCP
project ID. Pass it to Functions as `FIRESTORE_DATABASE_ID` and create clients with
`firestore.NewClientWithDatabase`; do not rely on the `(default)` database.

Cloud Storage is for generated JSON snapshots and future binary evidence files. Store file
metadata, ownership, review status, and permissions in Firestore.

## Snapshot Rules

Generate JSON snapshots after write operations:

- `SubmitJoin` writes Firestore documents and regenerates the member/account snapshot when
  a Firebase user is known.
- `SubmitVehicleBasics` writes the vehicle basics record and regenerates the member/account
  and public aggregate snapshots.
- `SubmitSOH` appends a battery reading and regenerates the member/account and public
  aggregate snapshots.
- Admin review or publish actions regenerate public aggregate statistics.

Member/account snapshots are private data. Store them in Firestore under
`memberSnapshots/{uid}` and, where useful for reduced read load, in a private Cloud Storage
object such as `members/{uid}.json`. Serve them only through `MemberData` after server-side
Firebase ID-token verification.

Public aggregate snapshots may be written to static JSON files or Cloud Storage objects
only after anonymisation, verification, and exclusion rules have been applied.

## Suggested Collections

- `members`
- `joinSubmissions`
- `vehicles`
- `batteryReadings` as the append-only SoH history; `vehicles.battery` may retain the latest
  reading for compatibility
- `evidenceFiles`
- `memberSnapshots`
- `publicStatsSnapshots`
- `auditEvents`

## Multiple Vehicles UX

The UX must assume a member can own, have owned, or help with more than one I-PACE.

- Member/account pages should show a list of vehicles, not a single vehicle panel.
- Vehicle data submission is an "add/register a vehicle" flow.
- After saving a vehicle, provide a route back to account/member dashboard and a clear
  option to add another vehicle.
- Public copy should avoid implying that each member has only one car.

## Privacy and Security

- Never store full VINs in Firestore, Cloud Storage, logs, static files, or test fixtures.
- Store `vinHash` using `VIN_PEPPER` and optionally `vinLast6` for member-facing reference.
- Never place names, emails, registrations, VIN fragments, or per-member vehicle details
  in public static JSON.
- Member/account snapshots must be served only after `MemberData` verifies a Firebase ID
  token server-side.

## Tests

Update or add tests for:

- raw VIN/full VIN fields are not introduced;
- member snapshots are private and not generated under public static output;
- Join and vehicle writes trigger member snapshot regeneration;
- vehicle writes update canonical Firestore documents and refresh the private snapshot;
- only the authenticated vehicle owner can append an SoH reading;
- repeated readings remain ordered and available in the member snapshot;
- admin/public stats generation excludes records marked out of public statistics.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Run `go test ./...` in `functions/firebase-go`.
- Confirm no private member/account JSON appears under `_site/`.
