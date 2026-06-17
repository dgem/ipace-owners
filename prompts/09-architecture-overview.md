# Architecture Overview Prompt

Read this prompt alongside the feature-specific prompts before making cross-layer changes.
It is the current source of truth for the I-PACE Owners' Advocacy Group architecture.

## Current target architecture

- **Static site:** Eleventy 3, Markdown/Nunjucks, custom CSS, no frontend framework.
- **Frontend JavaScript:** vanilla IIFEs loaded with `defer`; no bundler.
- **Authentication:** Firebase Authentication passwordless email links.
- **Backend:** Cloud Functions for Firebase / Google Cloud Functions, written in Go.
- **Canonical data:** Cloud Firestore.
- **Generated snapshots:** member/private and future public aggregate JSON written to
  Firestore and Cloud Storage so page loads avoid repeated canonical-store reads.
- **Hosting:** Firebase Hosting with rewrites from `/api/*` to Go Functions.
- **Infrastructure:** OpenTofu/Terraform under `infra/opentofu/`.
- **CI/CD:** GitHub Actions with GCP Workload Identity Federation. PRs deploy to staging
  Firebase Hosting preview channels; `main` deploys to production.
- **Domains/SSL:** Firebase Hosting managed SSL for `ipace-owners.org`; DNS remains at
  Fasthosts and must point to the records Firebase provides.

## Directory structure

```
src/                         Static Eleventy source
src/assets/js/identity.js    Firebase Auth email-link adapter
src/assets/js/member-auth.js Server-verified member/admin data loading
functions/firebase-go/       Go Cloud Functions
infra/opentofu/              GCP/Firebase infrastructure
firebase.json                Hosting headers, rewrites and Functions config
prompts/                     Sequenced rebuild/evolution prompts
```

Keep the legacy `netlify/` files only while migration is in progress. New work should target
Firebase/GCP unless explicitly maintaining the old deployment path.

## Authentication flow

1. Build-time Firebase web config is emitted by `.eleventy.js` from environment variables.
2. `identity.js` initialises Firebase Auth and never opens a password modal.
3. Magic-link forms call `POST /api/send-magic-link`.
4. The Go `SendMagicLink` Function calls Firebase Identity Toolkit to send an email sign-in
   link and returns account-enumeration-resistant `{ ok: true }` for valid email syntax.
5. When the user opens the email link, `identity.js` completes
   `signInWithEmailLink`, stores the session locally, clears auth query parameters, and
   exposes `window.ipaceGetIdentityToken()`.
6. Private API calls include `Authorization: Bearer <Firebase ID token>`.
7. Go Functions verify Firebase ID tokens server-side. Admin endpoints require an `admin`
   custom claim or a `roles` claim containing `admin`.

## Implemented API contracts

| Route | Function | Auth | Purpose |
|---|---|---|---|
| `POST /api/send-magic-link` | `SendMagicLink` | No | Send passwordless sign-in email link. |
| `POST /api/submit-join` | `SubmitJoin` | Optional | Save Join submission; send email link for guests. |
| `POST /api/submit-vehicle-basics` | `SubmitVehicleBasics` | Member | Save one vehicle basics record. Members may own multiple vehicles. |
| `GET /api/member-data` | `MemberData` | Member | Return the signed-in user's generated snapshot. |
| `GET /api/admin-data` | `AdminData` | Admin | Return review data for administrators. |

Firebase Hosting may also keep temporary rewrites for old `/.netlify/functions/*` paths
during migration, but templates and client JavaScript should use `/api/*`.

## Data model principles

- Firestore is canonical for structured owner, membership, vehicle, evidence, and review
  data.
- Cloud Storage is for generated JSON snapshots and future uploaded evidence blobs.
- Member pages read a generated member snapshot through `MemberData`; the Function verifies
  auth before returning it.
- Public dashboard pages should read anonymised aggregate JSON only after verification and
  exclusion rules have been applied.
- One member can have zero, one, or many vehicles; do not model the member account as a
  single-car profile.
- Full VINs are never stored. Store only an HMAC-SHA-256 digest using `VIN_PEPPER` plus the
  final six VIN characters for member reference.
- Raw email addresses and names must never appear in public static files or public
  aggregate JSON.

## Security constraints

- Verify every private request server-side with Firebase Admin SDK.
- Do not trust client-side `hidden` attributes or auth UI state.
- Do not log raw VINs, Identity tokens, full request bodies, or personal records.
- Return generic magic-link responses so account existence cannot be enumerated.
- Store secrets in GCP Secret Manager and GitHub environment secrets, never in git.
- Restrict CORS to production, staging preview hosts, and local development origins.
- Uploaded evidence must be validated server-side before storage.

## Infrastructure and deployment

- Use the shared OpenTofu module in `infra/opentofu/modules/ipace-owners` from the single
  environment root in `infra/opentofu/env`. Staging and production must use the same root
  and differ only by tfvars/input values.
- Required resources include Firebase project enablement, Firestore native mode, Cloud
  Web App config, Firestore native mode, Cloud Storage snapshot bucket, Secret Manager
  secrets, Function runtime service account, and GitHub Workload Identity Federation.
- The OpenTofu module should also bootstrap the GitHub Actions `staging` and `production`
  environments, including the variables and secrets consumed by the deploy workflows.
  Firebase web API keys, app IDs, auth domains and storage bucket values should be derived
  from resources created by OpenTofu, not pasted manually. Keep real secret values out of
  git; provide the remaining secret `VIN_PEPPER` through uncommitted tfvars or `TF_VAR_*`.
- GitHub Actions must run `npm test`, `npm run build`, and `go test ./...` for
  `functions/firebase-go` before deploying.
- PRs deploy to staging preview channels and run smoke tests against the published URL.
- Merges to `main` deploy production.

## Prompt maintenance

Keep these related prompts aligned when the architecture changes:

- `05-identity-member-admin-gating.md`
- `06-forms-and-evidence-collection.md`
- `08-backend-security-and-storage.md`
- `10-functions-shared-utilities.md`
- `11-functions-identity-and-join.md`
- `12-functions-vehicle-basics.md`
- `13-functions-member-admin-data.md`
- `14-functions-future-evidence-and-stats.md`
- `15-firestore-static-json-data-model.md`
