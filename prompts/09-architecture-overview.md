# Architecture Overview Prompt

Read this prompt alongside the feature-specific prompts before making cross-layer changes.
It is the current source of truth for the I-PACE Owners' Advocacy Group architecture.

## Current target architecture

- **Static site:** Eleventy 3, Markdown/Nunjucks, custom CSS, no frontend framework.
- **Frontend JavaScript:** vanilla IIFEs loaded with `defer`; no bundler.
- **Public presentation mode:** `site.defaultMode` controls the deployed `launch` or `full`
  experience. Synchronous `site-mode.js` runs before CSS, accepts `?site-mode=launch|full`, and
  persists valid overrides in session storage. Templates use `data-site-mode-only`; full-only
  content is hidden by default by a critical head style and the main stylesheet, and is only
  revealed when the root document is explicitly `data-site-mode="full"`. This only controls
  discoverability and never replaces server authorization.
- **Authentication:** Firebase Authentication passwordless email links.
- **Backend:** Cloud Functions for Firebase / Google Cloud Functions, written in Go.
- **Canonical data:** A named Cloud Firestore database per environment, with its database
  ID matching the GCP project ID. Go Functions select it explicitly using
  `firestore.NewClientWithDatabase`. Function environment generation may derive this ID
  from `FIREBASE_PROJECT_ID` during the initial OpenTofu/GitHub variable rollout. Production
  Firestore data is encrypted at rest with Google-managed encryption, has point-in-time
  recovery enabled, has Firestore delete protection and OpenTofu destroy prevention, and is
  covered by a daily Firestore backup schedule retained for 14 weeks. Staging intentionally
  does not carry those production-only backup/delete-protection settings.
- **Generated snapshots:** private member JSON written to Firestore and optionally Cloud
  Storage; public aggregate JSON written to Cloud Storage. Both are served through verified
  Functions rather than emitted into the static site.
- **Hosting:** Firebase Hosting with one `/api/**` rewrite to the Go `Api` Function.
- **Infrastructure:** OpenTofu/Terraform under `infra/opentofu/`.
- **CI/CD:** GitHub Actions with GCP Workload Identity Federation. PRs deploy to staging
  Firebase Hosting preview channels; `main` deploys to production.
- **Domains/SSL:** Firebase Hosting managed SSL for `ipace-owners.org`; DNS remains at
  Fasthosts. OpenTofu owns Firebase custom-domain associations, reports the required DNS
  records and validation state, while the records are entered manually in Fasthosts.
- **Authentication email delivery:** the shared OpenTofu module manages supported Identity
  Platform notification settings and custom sender-domain verification through a tested Admin
  v2 API bridge. Firebase's passwordless email-link body is fixed; versioned HTML
  account-action designs remain future assets. Branded passwordless delivery is implemented
  separately through server-generated Firebase links and Resend when configured.

## Directory structure

```
src/                         Static Eleventy source
src/assets/js/identity.js    Firebase Auth email-link adapter
src/assets/js/member-auth.js Server-verified member/admin data loading
functions/firebase-go/       Go Cloud Functions
infra/opentofu/              GCP/Firebase infrastructure
firebase.json                Firebase Hosting headers and rewrites
Makefile                     Shared local and CI command entrypoints
prompts/                     Sequenced rebuild/evolution prompts
```

Firebase/GCP is the only deployment and backend target. Do not add compatibility code for
the retired hosting or Function platform.

## Authentication flow

1. Build-time Firebase web config is emitted by `.eleventy.js` from environment variables.
2. `identity.js` initialises Firebase Auth and never opens a password modal.
3. Magic-link login forms call `POST /api/send-magic-link`.
4. The Go `Api` Function routes the request to `SendMagicLink`, which first checks for an
   existing Join submission matching the email fingerprint. It invokes the configured
   Firebase-default or Admin-SDK/Resend delivery path only for registered members and
   suppresses email side effects for unregistered addresses or lookup failures while
   returning account-enumeration-resistant `{ ok: true }` for valid email syntax. Set
   Identity Toolkit's `linkDomain` only for environments with a verified Firebase Hosting
   custom domain. If `FIREBASE_EMAIL_LINK_DOMAIN` is absent, derive it from the validated
   custom-domain `continueUrl`; never pass Firebase preview, default `web.app` /
   `firebaseapp.com`, localhost, or non-HTTPS hosts as `linkDomain`. Derive `continueUrl`
   from a validated request origin for allowed preview hosts; otherwise fall back to the
   environment account URL. When `RESEND_API_KEY` and `RESEND_FROM` are configured,
   generate the Firebase sign-in link server-side and send a branded Resend HTML/plain-text
   email using `/images/ipace-hero.png`; otherwise fall back to Firebase's default email
   sender.
5. When the user opens the email link, `identity.js` completes
   `signInWithEmailLink`, stores the session locally, clears auth query parameters, and
   exposes `window.ipaceGetIdentityToken()`.
6. Private API calls include `Authorization: Bearer <Firebase ID token>`.
7. Go Functions verify Firebase ID tokens server-side. Admin endpoints require an `admin`
   custom claim or a `roles` claim containing `admin`.

## Implemented API contracts

| Route | Handler behind `Api` | Auth | Purpose |
|---|---|---|---|
| `POST /api/send-magic-link` | `SendMagicLink` | No | Request a passwordless sign-in email for an already registered member. |
| `POST /api/submit-join` | `SubmitJoin` | Optional | Save Join submission; send email link for guests. |
| `POST /api/submit-vehicle-basics` | `SubmitVehicleBasics` | Member | Save one vehicle basics record and optional initial SoH reading. |
| `POST /api/submit-soh` | `SubmitSOH` | Member | Append an SoH reading after verifying vehicle ownership. |
| `POST /api/upsert-service-event` | `UpsertServiceEvent` | Member | Add or edit an owned vehicle's service/fault timeline record. |
| `GET /api/member-data` | `MemberData` | Member | Return the signed-in user's generated snapshot. |
| `GET /api/admin-data` | `AdminData` | Admin | Return review data for administrators. |
| `GET /api/public-stats` | `PublicStats` | No | Return the generated anonymised aggregate snapshot. |

Templates and client JavaScript use `/api/*`; Firebase Hosting rewrites `/api/**` to the
single Go `Api` Function, which dispatches to handler functions in process. `make
deploy-functions` deploys only `Api`; do not re-expand deployment to one Cloud Function per
route unless there is a measured need.

## Data model principles

- Firestore is canonical for structured owner, membership, vehicle, evidence, and review
  data.
- Cloud Storage is for generated JSON snapshots and future uploaded evidence blobs.
- Member pages read a generated member snapshot through `MemberData`; the Function verifies
  auth before returning it.
- Public dashboard pages read anonymised aggregate data through `PublicStats`. Aggregates
  must be generated from consent-filtered records and must not expose canonical member
  records, raw identifiers, registrations, VIN fragments, names, or emails.
- The public “Owners joined” counter is the number of contact-consenting Join submissions after
  lowercasing addresses and removing `+tag` aliases. `registeredMembers` remains the separate
  Firebase Auth account total in the aggregate schema.
- One member can have zero, one, or many vehicles; do not model the member account as a
  single-car profile.
- Store SoH measurements as append-only `batteryReadings` records tied to a vehicle. The
  embedded vehicle battery value is the latest compatibility value, not the historical
  source of truth.
- Store editable service and fault history in `serviceEvents`, tied to both the authenticated
  member UID and vehicle ID. Preserve creation timestamps and review metadata on edits.
- Regenerate private member snapshots after vehicle, SoH, or service-event writes. Regenerate
  public aggregate snapshots after Join, vehicle, and SoH writes, using only fields with
  defined consent and publication rules. Service/fault events stay private until explicit
  moderation and publication rules exist.
- Full VINs are never stored. Store only an HMAC-SHA-256 digest using `VIN_PEPPER` plus the
  final six VIN characters for member reference.
- Raw email addresses and names must never appear in public static files or public
  aggregate JSON.

## Security constraints

- Verify every private request server-side with Firebase Admin SDK.
- Do not trust client-side `hidden` attributes or auth UI state.
- Do not log raw VINs, Firebase ID tokens, full request bodies, or personal records.
- Return generic magic-link responses so account existence cannot be enumerated.
- Store secrets in GCP Secret Manager and GitHub environment secrets, never in git.
- Restrict CORS to production, staging preview hosts, and local development origins.
- Uploaded evidence must be validated server-side before storage.

## Infrastructure and deployment

- Use the shared OpenTofu module in `infra/opentofu/modules/ipace-owners` from the single
  environment root in `infra/opentofu/env`. Staging and production must use the same root
  and differ only by workspace plus tfvars/input values. Use the `staging` workspace with
  `staging.tfvars` and the `production` workspace with `production.tfvars`.
- The environment root should allow `project_id` to be omitted when creating a GCP project,
  deriving `${project_id_prefix}-${environment}` by default while still allowing an explicit
  project ID for existing projects or global ID collisions.
- Required resources include Firebase project enablement, Firestore native mode, Cloud
  Web App config, Firestore native mode, Cloud Storage snapshot bucket, Secret Manager
  secrets, Function runtime service account, and GitHub Workload Identity Federation.
- OpenTofu must configure Firebase Authentication / Identity Platform for passwordless
  email sign-in, with email sign-in enabled, password-required disabled, and authorized
  domains derived from `site_url` plus any explicit extra auth domains.
- Store future professional account-management email designs under
  `infra/opentofu/modules/ipace-owners/templates/auth-email`, but do not PATCH them while the
  product uses Firebase's passwordless email-link message. The Admin v2 account-management
  templates do not represent `EMAIL_SIGNIN`, and Firebase rejects those unrelated template
  fields in this configuration. The `terraform_data` bridge reconciles only supported locale
  and delivery settings plus sender-domain verification.
- Custom authentication sender domains require Firebase-issued TXT and CNAME records at
  Fasthosts. Verification is a two-stage `VERIFY` then `APPLY` operation and must be safely
  rerunnable with `make infra-email-domain ENV=<environment>`. Do not create a second SPF
  record; merge the Firebase include into the domain's existing SPF policy.
- Use `auth.stage.ipace-owners.org` for staging and `auth.ipace-owners.org` for production.
  Fasthosts remains the registrar, authoritative DNS host, and human-mail provider for launch.
  Sender-domain verification needs only TXT/CNAME additions at Fasthosts; it does not require
  nameserver, apex MX, mailbox, or Google Workspace changes. Treat any later Cloud DNS or mail
  migration as a separate project that first inventories every hosting and mail record.
- Firebase's built-in passwordless `EMAIL_SIGNIN` message body is not represented by the
  configurable Admin v2 account-management templates. Do not claim that its copy is managed
  by these HTML files. Fully branded magic-link copy requires generating the action link
  server-side and sending it through an explicitly selected transactional email or SMTP
  provider, with secrets managed outside git.
- The OpenTofu module should also bootstrap the GitHub Actions `staging` and `production`
  environments, including the variables and secrets consumed by the deploy workflows.
  Firebase web API keys, app IDs, auth domains and storage bucket values should be derived
  from resources created by OpenTofu, not pasted manually. Keep real secret values out of
  git; provide the remaining secret `VIN_PEPPER` through uncommitted tfvars or `TF_VAR_*`.
- The repository Makefile is the shared command surface for local development and CI.
  `make` and `make help` must print documented targets; `make functions` must list the
  Cloud Function entrypoints deployed by `make deploy-functions`. The expected production
  deployment surface is the single `Api` entrypoint.
- Infrastructure operations must use explicit `ENV=staging` or `ENV=production` Make
  targets. `make infra-plan` and `make deploy-hosting-env` should conditionally refresh
  gcloud user/ADC authentication, set an accessible ADC quota project, initialise the
  shared OpenTofu root, and select or create the workspace matching `ENV` before planning
  or applying the matching tfvars file. Never default an infrastructure apply to production.
- Define Firebase Hosting custom domains per environment without provider-side DNS waiting,
  because toggling that field forces replacement. Output both Hosting traffic/ownership and
  certificate ACME records. After those records are entered at Fasthosts, refresh and expose
  ownership, host and certificate state through `make infra-plan` and
  `make infra-dns-records`. Do not attempt unsupported Fasthosts API automation or modify
  unrelated email DNS records. Protect custom-domain resources from accidental deletion.
- GitHub Actions must delegate common operations through Make targets. Before deploying,
  run `make test-node`, `make test-go`, and `make build`; local verification can use
  `make test` and `make build`.
- GitHub Actions should use the current Node.js Active LTS from `.nvmrc`, Go 1.26 / `go126` for
  Cloud Run functions, and current runtime-compatible action majors. Deploy Cloud Function runtime
  environment variables from an env vars file rather than comma-separated `--set-env-vars`,
  because values such as `ALLOWED_ORIGINS` contain commas.
- Pin third-party Actions to immutable commit SHAs. Serialize both staging and production
  deployment jobs without cancellation, and regenerate the public statistics snapshot from
  Firebase Auth and Firestore during each authenticated deployment.
- Keep runtime, provider, dependency and action versions current. Use the latest production
  Active LTS for Node, latest GCP-supported Go runtime, current stable OpenTofu, latest
  compatible provider major, and latest compatible stable package releases. Commit lockfiles,
  run weekly Dependabot checks for npm, Go modules, Actions and OpenTofu, and require migration
  guide review plus full tests/build/provider validation for major updates.
- All PRs have a read-only validation job. Only PRs whose head branch belongs to this
  repository may request OIDC, staging secrets, or a staging deployment; external fork
  workflows require maintainer approval and stop after validation. Protect the GitHub
  `staging` environment with a required reviewer.
- Trusted same-repository PRs deploy to staging preview channels and run smoke tests directly
  in the staging workflow against the published URL. Do not rely on GitHub `deployment_status` events
  for smoke testing, because Firebase Hosting preview deployments do not consistently
  provide a usable site URL through those events.
  Production deploys should also run smoke tests directly after hosting deployment.
  Discover the generated preview URL, append its hostname to Firebase Auth's authorized
  domains, and run smoke tests against that URL. Deploy the Go `Api` Function only when
  backend-related files or Function environment generation change; otherwise reuse the
  existing staging `Api` revision. If `Api` is redeployed, refresh the preview channel so
  rewrites point at the current Function revision. Serialize staging deployments because
  preview channels share the staging Functions and Auth configuration. Do not depend on a
  staging custom domain for PR flows. Grant the GitHub deployer only `firebaseauth.configs.get` and
  `firebaseauth.configs.update` through a project custom role rather than an Identity Toolkit
  administrator role.
- Do not pass a Firebase Hosting preview/default `web.app` hostname as Identity Toolkit's
  `linkDomain`; omit that field for previews so Firebase uses its default action handler,
  while keeping the PR URL as `continueUrl`. Production uses its verified custom domain.
- Manage the Firebase project's public-facing display name through OpenTofu because
  Firebase's default Auth emails insert that value as `%APP_NAME%`. Production uses
  `I-PACE Owners`; staging uses `I-PACE Owners Staging`.
- Custom branded passwordless emails use Resend only when the API key and sender are
  configured in the Function environment. The Resend API key must be a GitHub environment
  secret; OpenTofu may bootstrap it from the sensitive `resend_api_key` variable when
  `bootstrap_resend_api_key_secret` is true, but should leave it alone when that boolean is
  false. Non-secret
  sender/reply-to/asset-base values may be managed as GitHub environment variables by
  OpenTofu. OpenTofu may also manage the Resend sending-domain resource and output DNS
  verification records; DNS remains manual while Fasthosts is authoritative.
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
- `17-operations-ci-and-troubleshooting.md`
- `20-clean-room-reconstruction-contract.md`
