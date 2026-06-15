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
- **Authentication:** Netlify Identity widget (`netlify-identity-widget` CDN).
- **Hosting:** Netlify (build command `npm run build`, publish directory `_site`).
- **Backend:** Netlify Functions + Netlify Blobs for Join submissions and vehicle basics.

### Directory structure

```
src/
  *.md / *.njk              — top-level page templates
  member/                   — member-gated placeholder pages
  admin/                    — admin-gated placeholder pages
  updates/                  — update/news posts (.md)
  _data/                    — global data (site.json, navigation.json)
  _includes/layouts/        — base.njk, page.njk, form-page.njk
  _includes/partials/       — header.njk, footer.njk, mobile-nav.njk, card.njk, callout.njk
  assets/css/site.css       — all styles
  assets/js/main.js         — mobile menu, nav current-page detection
  assets/js/identity.js     — Netlify Identity UI integration
  assets/js/multistep-form.js — generic multi-step form controller
public/images/              — static images (passed through to _site root)
netlify/functions/          — Netlify Functions
  lib/                      — shared Function utilities
    submission-utils.js     — CORS, origin allowlist, input sanitization, HMAC, Blobs
    identity-magic-link.js  — server-side Identity signup/recover flow
  send-magic-link.js        — standalone magic sign-in link request
  submit-join.js            — Join form: validate, store, send magic link
  submit-vehicle-basics.js  — signed-in vehicle basics: validate, store VIN HMAC
docs/architecture.md        — this document (living reference)
prompts/                    — sequenced prompts for rebuilding the product
.eleventy.js                — Eleventy configuration
netlify.toml                — build, redirect, header, and dev configuration
```

## Authentication Flow

### Netlify Identity

- The Identity widget is loaded from CDN in `base.njk` and initialised by `identity.js`.
- `identity.init()` uses `{ APIUrl: window.location.origin + '/.netlify/identity' }` so it
  works in all environments (no hard-coded domain).
- Owner accounts are created via the magic link flow (server-side signup/recover).
- Admin roles are assigned in `app_metadata.roles` via the Netlify Identity admin UI.
- Frontend gating uses `data-auth-gate`, `data-auth-content`, `data-admin-gate`, and
    `data-admin-content` attributes to show/hide content based on auth state.
- **Frontend gating is not access control.** Any Function that returns or processes private
  data must verify the Identity JWT server-side using `context.clientContext.user`.

### Magic Link Flow

1. Logged-out user submits Join form → browser calls `POST /.netlify/functions/submit-join`
   exactly once.
2. `submit-join` validates consent, stores the record in Netlify Blobs, then calls shared
   server-side magic-link code (`lib/identity-magic-link.js`) for logged-out users.
3. The shared code calls Netlify Identity `/signup` (new user) or `/recover` (existing user).
4. User clicks the link → signed in via Netlify Identity.
5. `identity.on('login')` fires → UI updates to show member content.

Existing users can request another magic link via `POST /.netlify/functions/send-magic-link`
without resubmitting the Join form.

## Form Handling and Storage

### Implemented Functions

| Function | Auth Required | What it does |
|---|---|---|
| `send-magic-link.js` | No | Standalone magic sign-in link request. Validates origin, email; calls Identity signup/recover server-side. Returns account-enumeration-resistant `{ ok }` flag. |
| `submit-join.js` | No (optional) | Validates Join form (consent required), stores membership interest in Blobs, sends magic link for logged-out users. |
| `submit-vehicle-basics.js` | Yes (signed-in user) | Validates vehicle identity and battery health slice, stores record in Blobs with VIN HMAC. Requires `VIN_PEPPER` env var. |

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
| **Member pages** (dashboard, my vehicle, account) | Client-side gated by Identity state. Future Functions return data only to the authenticated owner. | Frontend gating + server-side JWT verification on all data endpoints. |
| **Admin pages** (review queue, submissions) | Client-side gated by admin role. Future Functions verify admin role server-side before returning any data. | Frontend gating + server-side JWT + role check (`app_metadata.roles` contains `admin`). |

## Security Constraints

1. **Never store raw VINs, email addresses, or names in public static files.**
2. **Frontend role-checking is not access control.** Always verify JWTs server-side.
3. **Use HMAC (with pepper) for VIN deduplication**, not plain SHA-256 hashes.
4. **Netlify environment variables** must be used for all secrets (`VIN_PEPPER`, API tokens).
5. **CSP headers** are configured in `netlify.toml` to restrict script sources.
6. **Origin allowlist** in Functions rejects cross-site `no-cors` requests that could trigger
  unsolicited emails.
7. **Function logs** may include structured diagnostics (request IDs, methods, origins, status
  codes, email fingerprints) but must never log raw VINs, full personal records, Identity
  tokens, request bodies, or Identity response bodies.
8. **Uploads must be validated server-side** before storage — never trust client-supplied
  content types or filenames.

## Future Considerations

- **Netlify Database / Postgres** — if querying across submissions becomes important
   (filtering by model year, SoH range), a relational database may replace or supplement Blobs.
- **Email notifications** — Netlify Functions can trigger emails via transactional providers
   (Postmark, SendGrid) for submission confirmations and status updates.
- **Evidence document uploads** — Blobs can store binary files. Requires server-side
  validation of file type, size, and content before storage.
- **Proposed Functions:** `submit-vehicle.js` (full evidence), `get-member-data.js` (member's
  own submissions), `admin-review.js` (pending submissions for admins),
    `admin-update-submission.js` (review status changes), `public-stats.js` (aggregate data).

## Validation

- Run `npm run build` to confirm the site builds cleanly.
- Run `npm test` to confirm all Function tests pass.
- Confirm no real owner data, raw VINs, or personal information appears in static output.
- Confirm `docs/architecture.md` stays in sync with this prompt when the architecture evolves.
