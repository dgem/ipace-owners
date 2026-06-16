# Postgres and Static JSON Data Model Prompt

Use this prompt when migrating storage from Netlify Blobs to Netlify Database/Postgres or
when changing generated JSON snapshots for member/account pages or public evidence stats.

## Goal

Use Postgres as the canonical source for all structured data. Use generated JSON snapshots
to reduce read load and keep public pages static/CDN-friendly.

## Canonical Data Store

Netlify Database/Postgres is the source of truth for:

- members and Identity linkage;
- Join submissions, relationship status, skills, and consents;
- vehicles, with one member able to own or submit data for multiple vehicles;
- battery State of Health readings;
- future recall, repair, loan car, payment, goodwill, responsibility, and review data;
- admin review state, verification levels, and audit events.

Netlify Blobs should be used for binary evidence files only. Store file metadata,
ownership, review status, and permissions in Postgres.

## Static JSON Snapshots

Generate JSON snapshots after write operations:

- `submit-join` writes Postgres rows and regenerates the member/account snapshot.
- `submit-vehicle-basics` writes or updates the relevant vehicle rows and regenerates the
  member/account snapshot.
- Admin review or publish actions regenerate public aggregate statistics.

Member/account snapshots are private data. Do not write them into public static output.
Store them in Postgres (`member_static_snapshots`) or another private store and serve them
only through `member-data.js` after server-side Identity verification.

Public aggregate snapshots may be written to static JSON files such as:

- `/assets/data/public-evidence-summary.json`
- `/assets/data/public-dashboard.json`

Public snapshots must contain only anonymised aggregate data and must exclude records with
review status or verification level indicating exclusion from public statistics.

## Required Tables

Maintain migrations under `netlify/database/migrations/`.

The initial schema should include:

- `members`
- `join_submissions`
- `vehicles`
- `vehicle_battery_readings`
- `evidence_files`
- `member_static_snapshots`
- `public_stats_snapshots`
- `audit_events`

## Multiple Vehicles UX

The UX must assume a member can own, have owned, or help with more than one I-PACE.

- Member/account pages should show a list of vehicles, not a single vehicle panel.
- Vehicle data submission is an "add/register a vehicle" flow.
- After saving a vehicle, provide a route back to account/member dashboard and a clear
  option to add another vehicle.
- Public copy should avoid implying that each member has only one car.

## Privacy and Security

- Never store full VINs in Postgres, Blobs, logs, static files, or test fixtures.
- Store `vin_hmac` using `VIN_PEPPER` and optionally `vin_last6` for member-facing
  reference.
- Never place names, emails, registrations, VIN fragments, or per-member vehicle details
  in public static JSON.
- Member/account snapshots must be served only after `member-data.js` verifies the
  Identity JWT server-side.

## Tests

Update or add tests for:

- migration schema contains expected tables and indexes;
- raw VIN/full VIN columns are not introduced;
- member snapshots are private and not generated under public static output;
- Join and vehicle writes trigger member snapshot regeneration;
- admin/public stats generation excludes records marked out of public statistics.

## Validation

- Run `npm test`.
- Run `npm run build`.
- Confirm no private member/account JSON appears under `_site/`.
