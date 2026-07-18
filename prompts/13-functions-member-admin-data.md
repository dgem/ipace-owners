# Member and Admin Data Functions Prompt

Use this prompt when changing:

- `functions/firebase-go` `MemberData` or `AdminData`
- `src/assets/js/member-auth.js`
- member dashboard/member/account/admin review queue data rendering

## Goal

Serve private member/account snapshots and admin review data only through server-verified
Go Cloud Functions. Do not rely on client-side gating for private data access.

## Current flow

- `member-auth.js` loads on pages with `[data-auth-container]` or `[data-admin-container]`.
- For member pages, it fetches `GET /api/member-data`.
- For admin pages, it fetches `GET /api/admin-data`.
- Content is hidden by default and is revealed only after the relevant Function returns 200.
- Login/admin gates remain visible for 401 or 403 responses.
- Firestore is canonical. Member/account pages should be served from a private generated
  JSON snapshot regenerated during Join signup, vehicle add/update, SoH, and service-event flows.
- `MemberData` should read the private generated snapshot first, then regenerate it from
  Firestore if missing.

## MemberData

- Requires a valid Firebase ID token.
- Returns only the authenticated user's own private member/account snapshot.
- The snapshot may include Join information, zero or more vehicles, append-only SoH readings,
  and editable service/fault records.
- Do not serve member snapshots from public static files. Return them only after
  server-side Firebase verification.
- Does not expose admin-only review data to members.

Member responses may include:

- `identityUserId`
- `email`
- `joinRecords`
- `vehicleRecords`
- `batteryReadings`
- `serviceEvents`

## AdminData

- Requires a valid Firebase ID token.
- Requires an `admin` custom claim or `roles` containing `admin`.
- Returns 401 for unauthenticated users.
- Returns 403 for authenticated non-admin users.
- Reads Firestore review records.
- Returns Join, member, vehicle, and vehicle-basics records for review.
- Admin responses may include review state and `userEmailHash`.

## Browser rendering

`member-auth.js` populates shared account fields and dispatches a `member:data` event so the
dashboard workspace can render server-verified data. Browser rendering may populate:

- `[data-vehicle-list]`
- `[data-join-info]`
- `[data-admin-stats]`
- `[data-admin-join-table]`
- `[data-admin-vehicle-table]`
- `[data-vehicle-workspace]`

Rendering must not leak private data before a successful Function response. Avoid placing
private data directly in static HTML.

## Tests

Required coverage:

- Member endpoint rejects unauthenticated users.
- Member endpoint returns only the current user's records.
- Member endpoint supports multiple vehicle records for one user.
- Account pages should present vehicles as a full-width summary section, not a narrow
  column. Keep detailed SoH and service/fault editing in the member dashboard workspace,
  and link account vehicle summaries to that workspace.
- Member endpoint can serve the private generated snapshot without scanning canonical
  records on every page load.
- Admin endpoint rejects unauthenticated users.
- Admin endpoint rejects authenticated non-admin users.
- Admin endpoint returns review-capable records for admin users.
- Disallowed origins are rejected before storage reads.

## Validation

- Run `make test`.
- Run `make build`.
- Run `GOCACHE=/tmp/ipace-owners-go-build make test-go` or `go test ./...` in `functions/firebase-go`.
