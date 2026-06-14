# Architecture — I-PACE Owners' Advocacy Group

## Overview

This document describes the intended future architecture for the I-PACE Owners' Advocacy
Group platform. The current PR establishes the static site foundation; backend features
will be added incrementally.

## Current state (MVP)

- **Eleventy** static site generator (v3)
- **Nunjucks** templates with Markdown-authored content
- **Custom CSS** (no frameworks)
- **Netlify Identity** widget for frontend authentication
- **Netlify** hosting and deployment
- **Netlify Forms** for Join membership expressions of interest
- **No vehicle/evidence persistence yet** — vehicle submissions and evidence uploads are not stored

## Intended future architecture

### Authentication — Netlify Identity

Netlify Identity provides JWT-based authentication. The Identity widget is loaded from the
CDN and initialised on every page.

- Owner accounts are created via Netlify Identity (email/password)
- Admin roles are assigned in `app_metadata.roles` via the Netlify Identity admin UI or API
- The frontend checks Identity state to gate member and admin UI
- **Frontend gating is not sufficient for real private data** — all Netlify Functions that
  return or process private data must verify the Identity JWT server-side

### Form handling

The Join form is configured as a Netlify Form named `join`. JavaScript-enhanced submissions
post URL-encoded form data to Netlify Forms while keeping the multi-step completion screen
visible. Without JavaScript, the Join form submits as a normal HTML form.

Vehicle/evidence forms currently prevent default submission and show a placeholder message.

### Future form handling — Netlify Functions

When backend integration is added:
- Vehicle/evidence forms POST to a Netlify Function endpoint (for example,
  `/.netlify/functions/submit-vehicle`)
- The Join form may later move from Netlify Forms to a Function if membership intake needs
  Identity-linked profiles, stronger validation, or richer admin workflows
- The Function verifies the Identity JWT from the `Authorization` header
- Validated data is stored in Netlify Blobs (JSON record) and/or Netlify Database
- The Function returns a confirmation response

Example function structure:
```
netlify/functions/
  submit-join.js        — optional future replacement for Netlify Forms
  submit-vehicle.js     — handles vehicle data submissions
  get-member-data.js    — returns member's own submissions (auth required)
  admin-review.js       — returns pending submissions (admin auth required)
```

### Data storage — Netlify Blobs

Netlify Blobs provides key-value object storage suitable for JSON records.

- Each vehicle data submission stored as a JSON blob keyed by submission ID
- Owner metadata stored separately (linked by hashed identifier)
- Uploaded evidence documents stored as separate blobs (future)
- Aggregate statistics computed server-side and cached as a separate blob

### VIN deduplication

Full VINs should never be stored in plain text in public static files or logs.

For deduplication, use an **HMAC with a secret pepper** (stored as a Netlify environment
variable), not a plain SHA-256 hash. This prevents rainbow table attacks against a
compromised blob store.

```javascript
// Example (pseudocode — not implemented in this PR)
const crypto = require('crypto');
const vinHash = crypto
  .createHmac('sha256', process.env.VIN_PEPPER)
  .update(vin.toUpperCase().trim())
  .digest('hex');
```

### Public vs private data

- **Public pages** (evidence dashboard, methodology, FAQ) — serve only anonymised aggregate
  statistics. No individual records, no VINs, no personal data.
- **Member pages** (dashboard, my vehicle, account) — served client-side gated by Identity
  state. Future Functions will return data only to the authenticated owner.
- **Admin pages** (review queue, submissions) — served client-side gated by admin role.
  Future Functions will verify the admin role server-side before returning any data.

### Future considerations

- **Netlify Database / Postgres** — if querying across submissions becomes important
  (e.g. filtering by model year, SoH range), a relational database may be added.
- **Email notifications** — Netlify Functions can trigger emails via a transactional
  provider (e.g. Postmark, SendGrid) for submission confirmations and updates.
- **Evidence document uploads** — Blobs can store binary files. Uploads require server-side
  validation of file type, size, and content before storage.

## Security notes

1. **Never store raw VINs, email addresses, or names in public static files.**
2. **Frontend role-checking is not access control.** Always verify JWTs server-side.
3. **Use HMAC (with pepper) for VIN deduplication**, not plain hashes.
4. **Netlify environment variables** must be used for all secrets (HMAC keys, API tokens).
5. **CSP headers** are configured in `netlify.toml` to restrict script sources.
6. **Uploads must be validated server-side** before storage — never trust client-supplied
   content types or filenames.
