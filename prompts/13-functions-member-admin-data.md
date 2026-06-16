# Member and Admin Data Functions Prompt

Use this prompt when changing:

- `netlify/functions/member-data.js`
- `netlify/functions/admin-data.js`
- `src/assets/js/member-auth.js`
- member dashboard/account/admin review queue data rendering

## Goal

Serve private member/account snapshots and admin review data only through server-verified
Netlify Identity Functions. Do not rely on client-side gating for private data access.

## Current Flow

- `member-auth.js` loads on pages with `[data-auth-container]` or `[data-admin-container]`.
- For member pages, it fetches `/.netlify/functions/member-data`.
- For admin pages, it fetches `/.netlify/functions/admin-data`.
- Content is hidden by default and is revealed only after the relevant Function returns 200.
- Login/admin gates remain visible for 401 or 403 responses.
- Postgres is the canonical source for structured data. Member/account pages should be
  served from a private generated JSON snapshot regenerated during Join signup and vehicle
  add/update flows.

## `member-data.js`

- Requires `context.clientContext.user.sub`.
- Returns only the authenticated user's own private member/account snapshot.
- The snapshot may include Join information and a list of zero, one, or many vehicles.
- Do not serve member snapshots from public static files. They must be returned only after
  server-side Identity verification.
- Does not expose admin-only review data to members.

Member responses may include:

- `identityUserId`
- `email`
- `joinRecords`
- `vehicleRecords`

## `admin-data.js`

- Requires `context.clientContext.user.sub`.
- Requires `admin` in `user.app_metadata.roles`.
- Returns 401 for unauthenticated users.
- Returns 403 for authenticated non-admin users.
- Returns Join, member, vehicle, and vehicle-basics records for review.
- Admin responses may include review state and `userEmailHash`.

## Browser Rendering

`member-auth.js` may populate:

- `[data-vehicle-list]`
- `[data-join-info]`
- `[data-admin-stats]`
- `[data-admin-join-table]`
- `[data-admin-vehicle-table]`

Rendering must not leak private data before a successful Function response. Avoid placing
private data directly in static HTML.

## Tests

Update:

- `test/member-data.test.js`
- `test/admin-data.test.js`

Required coverage:

- Member endpoint rejects unauthenticated users.
- Member endpoint returns only the current user's records.
- Member endpoint supports multiple vehicle records for one user.
- Admin endpoint rejects unauthenticated users.
- Admin endpoint rejects authenticated non-admin users.
- Admin endpoint returns review-capable records for admin users.
- Disallowed origins are rejected before storage reads.

## Validation

- Run `npm test`.
- Run `npm run build`.
