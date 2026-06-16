# Architecture Overview Prompt

Read this prompt alongside the feature-specific prompts to understand the full system
architecture, data flows, and security constraints for the I-PACE Owners' Advocacy Group
platform.

## Goal

Ensure every agent has a complete picture of how the pieces fit together — static site,
authentication, form handling, storage, and security — before making changes that affect
more than one layer.

## Current Architecture (MVP)

### Stack

- **Static site generator:** Eleventy 3 (`@11ty/eleventy`) with CommonJS config (`.eleventy.js`).
- **Templates:** Nunjucks (`.njk`) for complex pages; Markdown (`.md`) for content pages.
- **CSS:** Custom CSS only — no Tailwind, Bootstrap, or utility frameworks. All styles in
     `src/assets/css/site.css` (reset → tokens → typography → layout → components → print).
- **JavaScript:** Plain vanilla JS — no React, Vue, Svelte, TypeScript, or bundler. Each
  file is an IIFE loaded with `defer`.
- **Authentication:** Netlify Identity widget (`netlify-identity-widget` CDN) + server-side JWT verification via Netlify Functions.
- **Hosting:** Netlify (build command `npm run build`, publish directory `_site`).
- **Backend:** Netlify Functions + Netlify Blobs for Join submissions and vehicle basics.

### Directory structure

```
src/
   *.md / *.njk               — top-level page templates
   member/                    — member-gated pages (server-side auth)
   admin/                     — admin-gated pages (server-side auth)
   updates/                   — update/news posts (.md)
    _data/                    — global data (site.json, navigation.json)
    _includes/layouts/        — base.njk, page.njk, form-page.njk
    _includes/partials/       — header.njk, footer.njk, mobile-nav.njk, card.njk, callout.njk
   assets/css/site.css        — all styles
   assets/js/main.js          — mobile menu, nav current-page detection
   assets/js/identity.js      — Netlify Identity UI (header only, no content gating)
   assets/js/member-auth.js   — server-side auth verification and data population
   assets/js/multistep-form.js — generic multi-step form controller
public/images/               — static images (passed through to _site root)
netlify/functions/           — Netlify Functions
   lib/                       — shared Function utilities
     submission-utils.js      — CORS, origin allowlist, input sanitization, HMAC, Blobs
     identity-magic-link.js   — server-side Identity signup/recover flow
   send-magic-link.js         — standalone magic sign-in link request
   submit-join.js             — Join form: validate, store, send magic link
   submit-vehicle-basics.js   — signed-in vehicle basics: validate, store VIN HMAC
   member-data.js             — authenticated user's own records (server-side JWT check)
   admin-data.js              — all records for admins (server-side JWT + role check)
prompts/                     — sequenced prompts for rebuilding the product
.eleventy.js                 — Eleventy configuration
netlify.toml                 — build, redirect, header, and dev configuration
```

## Authentication Flow

### Netlify Identity

- The Identity widget is loaded from CDN in `base.njk` and initialised by `identity.js`.
- `identity.init()` uses `{ APIUrl: window.location.origin + '/.netlify/identity' }` so it
  works in all environments (no hard-coded domain).
- Owner accounts are created via the magic link flow (server-side signup/recover).
- Admin roles are assigned in `app_metadata.roles` via the Netlify Identity admin UI.

### Server-Side Auth Verification

**Client-side gating has been removed.** All data access is now verified server-side:

1. `member-auth.js` loads on every page (via `base.njk`).
2. On DOM ready, it finds `[data-auth-container]` or `[data-admin-container]`.
3. It fetches the appropriate Function (`member-data` or `admin-data`) with browser's Identity cookie.
4. The Function verifies `context.clientContext.user` server-side (JWT validation by Netlify runtime).
5. On 200: `member-auth.js` hides the gate, shows content, and populates data from the response.
6. On 401/403: the gate stays visible — no private data is exposed.

**No client-side attribute manipulation can bypass auth.** The login gate is shown by default (`hidden` is on the content, not the gate). Content is only revealed after the server confirms authorization.

## Form Handling and Storage

### Implemented Functions

| Function | Auth Required | What it does |
|---|---|---|
| `send-magic-link.js` | No | Standalone magic sign-in link request. Validates origin, email; calls Identity signup/recover server-side. Returns account-enumeration-resistant `{ ok }` flag. |
| `submit-join.js` | No (optional) | Validates Join form (consent required), stores membership interest in Blobs, sends magic link for logged-out users. |
| `submit-vehicle-basics.js` | Yes (signed-in user) | Validates vehicle identity and battery health slice, stores record in Blobs with VIN HMAC. Requires `VIN_PEPPER` env var. |
| `member-data.js` | Yes (signed-in user) | Returns the authenticated user's own join and vehicle-basics records. Verifies `context.clientContext.user.sub`. Does not expose review data to members. |
| `admin-data.js` | Yes (admin role) | Returns all join and vehicle-basics records. Verifies JWT + `app_metadata.roles` contains `admin`. Exposes review data and userEmailHash to admins only. |

### Storage Model (Netlify Blobs)

- Join records: `join/<submission-id>.json`
- Vehicle basics: `vehicle-basics/<identity-user-id>/<submission-id>.json`
- Submission IDs use format `<prefix>_<uuid>` (e.g., `join_abc123`).
- Metadata is attached to each blob for future querying/filtering.

### VIN Deduplication

- Full VINs are never stored. The Function creates an HMAC-SHA-256 using `VIN_PEPPER` and
  stores only the HMAC digest plus the final six characters of the VIN for reference.
- If a VIN is provided and `VIN_PEPPER` is missing, reject the write entirely — do not
  store a weak hash or the full VIN.

## Public vs Private Data Boundaries

| Layer | What it serves | Access control |
|---|---|---|
| **Public pages** (evidence dashboard, methodology, FAQ) | Anonymised aggregate statistics only. No individual records, no VINs, no personal data. | None — served as static HTML. |
| **Member pages** (dashboard, account, submit vehicle data) | Server-side verified via `member-data.js`. Shows user's own join info and vehicle records. Login gate shown by default; content revealed only after Function returns 200. | Server-side JWT verification in `member-data.js`. No client-side gating — `member-auth.js` relies entirely on Function response. |
| **Admin pages** (review queue, submissions) | Server-side verified via `admin-data.js`. Shows all join and vehicle records with review data. Login gate + admin-only gate shown by default; content revealed only after Function returns 200. | Server-side JWT + role check in `admin-data.js` (`app_metadata.roles` contains `admin`). Returns 403 for non-admin users. |

## Security Constraints

1. **Never store raw VINs, email addresses, or names in public static files.**
2. **All data access is server-side verified.** `member-data.js` and `admin-data.js` verify Identity JWTs via `context.clientContext.user`. No private data is accessible without a valid JWT.
3. **Client-side gating is cosmetic only.** `identity.js` does not show/hide gated content. `member-auth.js` fetches from Functions and only reveals content on 200 response. Login gates are shown by default (content has `hidden` attribute).
4. **Use HMAC (with pepper) for VIN deduplication**, not plain SHA-256 hashes.
5. **Netlify environment variables** must be used for all secrets (`VIN_PEPPER`, API tokens).
6. **CSP headers** are configured in `netlify.toml` to restrict script sources.
7. **Origin allowlist** in Functions rejects cross-site `no-cors` requests that could trigger
  unsolicited emails.
8. **Function logs** may include structured diagnostics (request IDs, methods, origins, status
  codes, email fingerprints) but must never log raw VINs, full personal records, Identity
  tokens, request bodies, or Identity response bodies.
9. **Uploads must be validated server-side** before storage — never trust client-supplied
  content types or filenames.

## Future Considerations

- **Netlify Database / Postgres** — if querying across submissions becomes important
   (filtering by model year, SoH range), a relational database may replace or supplement Blobs.
- **Email notifications** — Netlify Functions can trigger emails via transactional providers
   (Postmark, SendGrid) for submission confirmations and status updates.
- **Evidence document uploads** — Blobs can store binary files. Requires server-side
  validation of file type, size, and content before storage.
- **Proposed Functions:** `submit-vehicle.js` (full evidence), `admin-update-submission.js` (review status changes), `public-stats.js` (aggregate data).

## Component Prompts

Use the smaller backend prompts for implementation details:

- `08-backend-security-and-storage.md` — shared backend security and storage constraints.
- `10-functions-shared-utilities.md` — shared Function utilities.
- `11-functions-identity-and-join.md` — magic links and Join submission.
- `12-functions-vehicle-basics.md` — signed-in vehicle basics.
- `13-functions-member-admin-data.md` — member/admin data reads.
- `14-functions-future-evidence-and-stats.md` — future evidence, uploads, review mutation, and public statistics.

## Validation

- Run `npm run build` to confirm the site builds cleanly.
- Run `npm test` to confirm all Function tests pass.
- Confirm no real owner data, raw VINs, or personal information appears in static output.
- Keep this prompt aligned with README, AGENTS.md, and the component prompts when the
  architecture evolves.
