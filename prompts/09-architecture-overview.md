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
4. The Go `Api` Function routes the request to `SendMagicLink`, which checks for an existing
   Join submission matching the email fingerprint and falls back to an exact Firebase
   Authentication email lookup. It invokes the configured Firebase-default or
   Admin-SDK/Resend delivery path only for registered members or existing Auth accounts and
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

OpenTofu reconciles the authoritative Firebase administrator emails through a tested Identity
Platform API bridge. The shared module always includes `dan@kanzi.co.uk`, resolves email to the
environment-specific UID, preserves unrelated custom claims, and revokes only admin access from
users removed from the desired set. A configured user must already exist in Firebase Auth.

After claims are verified, render one Admin action in the primary desktop controls and one in the
mobile Member section, both linking to `/admin/`. The admin dashboard is the sole shared directory
of admin tools; do not duplicate its tool links in header navigation.
For signed-out mobile visitors, keep Sign in visible beside the menu toggle as well as inside
the drawer, then hide both signed-out actions when authentication succeeds. The mobile header
action must stay hidden at desktop widths even after the shared `.btn` display rule is applied.
On member and admin routes, use compact one-line `Member › current page` and
`Admin › current page` secondary breadcrumb rows with the current crumb visibly emphasised.
Admin tool breadcrumbs link their section crumb to `/admin/`; dashboard and page headings do
not repeat a redundant Admin eyebrow. The mobile drawer exposes My Data and Add Vehicle. The
primary `My Data` action links to `/member/account/`, and the member email is not shown in the
header.

## Implemented API contracts

| Route | Handler behind `Api` | Auth | Purpose |
|---|---|---|---|
| `POST /api/send-magic-link` | `SendMagicLink` | No | Request a passwordless sign-in email for an already registered member. |
| `POST /api/submit-join` | `SubmitJoin` | Optional | Save Join submission; send email link for guests. |
| `POST /api/submit-vehicle-basics` | `SubmitVehicleBasics` | Member | Create or edit one owned vehicle basics record; creation may include an initial SoH reading. |
| `POST /api/submit-soh` | `SubmitSOH` | Member | Create or edit an owned SoH reading after verifying vehicle ownership. |
| `POST /api/upsert-service-event` | `UpsertServiceEvent` | Member | Add or edit an owned vehicle's service/fault timeline record. |
| `POST /api/update-member-preferences` | `UpdateMemberPreferences` | Member | Update contact and anonymised-analysis consent for all Join records of the signed-in email. |
| `POST /api/delete-vehicle` | `DeleteVehicle` | Member | Soft-delete an owned vehicle and its dependent SoH and service records after typed confirmation. |
| `POST /api/delete-soh` | `DeleteSOH` | Member | Soft-delete an owned SoH reading after typed confirmation. |
| `POST /api/delete-service-event` | `DeleteServiceEvent` | Member | Soft-delete an owned service/fault record after typed confirmation. |
| `GET /api/member-data` | `MemberData` | Member | Return the signed-in user's generated snapshot. |
| `GET /api/member-export?format=csv\|xlsx` | `MemberExport` | Member | Download that snapshot as separate CSV datasets in a ZIP or a formatted Excel workbook. |
| `GET /api/admin-data` | `AdminData` | Admin | Return review data for administrators. |
| `GET /api/admin/stats` | `AdminStats` | Admin | Return public consent-filtered homepage counters alongside private all-record member, vehicle, SoH, and service-event statistics with `Cache-Control: private, no-store`; canonical emails are deduplicated at the first Join before the daily Join trend and country rows are calculated. Derive country from the Join, one unambiguous vehicle country, or a strict UK registration, otherwise use `Unknown` (including conflicting vehicle countries). Magic-link-verified accounts have a separate daily line chart, and no per-vehicle evidence is returned. |
| `POST /api/admin/reengagement-preview` | `AdminReengagementPreview` | Admin | Return aggregate counts for the consented, unsigned-in Join audience. |
| `POST /api/admin/reengagement-send` | `AdminReengagementSend` | Admin | Confirm and send the next resumable batch of at most ten reminders. |
| `POST /api/admin/member-referral-preview` | `AdminMemberReferralPreview` | Admin | Preview the consented registered-member referral audience and exact campaign copy. |
| `POST /api/admin/member-referral-send` | `AdminMemberReferralSend` | Admin | Confirm and send the next resumable batch of at most ten referral emails. |
| `POST /api/admin/all-members-drive-preview` | `AdminAllMembersDrivePreview` | Admin | Preview the contact-consenting, canonical-email-deduplicated audience across verified and unverified Join records. |
| `POST /api/admin/all-members-drive-send` | `AdminAllMembersDriveSend` | Admin | Confirm and send the next resumable batch of at most ten all-member recruitment emails. |
| `POST /api/admin/jlr-contact-preview` | `AdminJLRContactPreview` | Admin | Load the fixed JLR Contact Markdown source, calculate the verified, consented audience, and return the exact branded preview. |
| `POST /api/admin/email-campaign-history` | `AdminEmailCampaignHistory` | Admin | Return campaign metadata and aggregate delivery history without recipient addresses. |
| `POST /api/admin/custom-campaign-preview` | `AdminCustomCampaignPreview` | Admin | Validate and persist a custom Markdown draft, calculate the verified consented audience, and return personalised HTML/plain-text previews. |
| `POST /api/admin/custom-campaign-send` | `AdminCustomCampaignSend` | Admin | Recheck the saved draft and audience, require exact confirmation, and send the next idempotent batch of at most ten. |
| `POST /api/admin/instagram-preview` | `AdminInstagramPreview` | Admin | Validate and preview an administrator-reviewed, chat-prepared Reel path and caption without publishing. |
| `POST /api/admin/instagram-campaign-history` | `AdminInstagramCampaignHistory` | Admin | List saved drafts and publication records and refresh cached provider insights when available. |
| `POST /api/admin/campaign-summary` | `AdminCampaignSummary` | Admin | Aggregate email delivery, Instagram publication/insight, and Facebook integration-availability totals for the Admin home. |
| `GET/POST/PUT/DELETE /api/admin/surveys` | `AdminSurveys` | Admin | Create, edit, list, and remove timed member surveys. |
| `GET /api/admin/survey-results` | `AdminSurveyResults` | Admin | Return aggregate and individual survey responses for manual analysis, including optional preferred-option counts; `format=csv` exports masked, PII-minimised rows. |
| `GET /api/member/surveys` | `MemberSurveys` | Member | List surveys, the member's own response, and permitted aggregate counts. |
| `POST /api/member/survey-response` | `SubmitSurveyResponse` | Member | Validate and save the member's one replaceable survey response. |
| `POST /api/admin/instagram-publish` | `AdminInstagramPublish` | Admin | Revalidate the exact draft and confirmation, process the Reel through Meta, and publish it immediately. |
| `POST /api/admin/instagram-generate` | `AdminInstagramGenerate` | Admin | Idempotently start the explicitly confirmed, billable eight-second Veo generation operation. |
| `POST /api/admin/instagram-generation-status` | `AdminInstagramGenerationStatus` | Admin | Poll and advance the job into the seven-second continuation, promote the 15-second master, and issue a short-lived review path. |
| `GET/HEAD /api/instagram-media/**` | `InstagramGeneratedMedia` | Expiring bearer path | Range-stream a private completed master after constant-time token and expiry validation. |
| `GET /api/public-stats` | `PublicStats` | No | Return the generated anonymised aggregate snapshot. |

Templates and client JavaScript use `/api/*`; Firebase Hosting rewrites `/api/**` to the
single Go `Api` Function, which dispatches to handler functions in process. `make
deploy-functions` deploys only `Api`; do not re-expand deployment to one Cloud Function per
route unless there is a measured need.

Member exports are assembled in memory from the authenticated private snapshot and returned
with private, no-store response headers. The separately committed
`public/downloads/sample-ipace-owner-data.xlsx` is deliberately public, contains fictional
records only, and is generated through the same workbook builder to demonstrate the real
sheet and chart format without exposing member data.

## Data model principles

- Firestore is canonical for structured owner, membership, vehicle, evidence, and review
  data.
- Cloud Storage is for generated JSON snapshots and future uploaded evidence blobs.
- Member pages read a generated member snapshot through `MemberData`; the Function verifies
  auth before returning it.
- Member exports are generated in memory from the same authenticated snapshot, exclude
  internal UID and hash fields, neutralise spreadsheet formula injection, and use private
  no-store attachment responses.
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
  Service-provider references retain the Jaguar locator CI code, provider name and postcode
  plus the member-confirmed authorised-JLR flag. The member lookup must search provider name,
  town, postcode, county, and address fragments, while related campaigns use a compact,
  overflow-safe wrapping grid. Also retain parts-delay range, goodwill payment and miles driven
  whilst faulty. Derive days to final fix server-side from the two recorded dates.
- Store named Instagram drafts and publication records in `instagramCampaigns`. Drafts can be
  updated, but published records are immutable; an edited repost receives a new ID and
  `sourceCampaignId`. Reserve before contacting Meta; a published retry returns the existing
  media ID. Cache supported provider insights without making history depend on their availability.
- Store Veo generation state in `instagramGenerationJobs`, keyed by a hash of the browser request
  ID. Keep generation and publication ledgers separate. A job progresses from initial generation
  through a transactionally claimed seven-second Veo video extension to a private versioned
  master. Concurrent status polls must not duplicate either billable provider operation, and
  generation never publishes as a side effect.
- Regenerate private member snapshots after vehicle, SoH, or service-event writes. Regenerate
  public aggregate snapshots after Join, vehicle, SoH, and service-event writes, using only
  records with anonymised-analysis consent and excluding records marked out of public reporting.
  Publish only the aggregate service/fault record count; service-event details remain private.
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
- Keep Instagram access tokens server-side. Accept only site-relative MP4/MOV paths, construct
  provider-fetchable URLs from the configured HTTPS media origin, and require an explicit
  full-video review plus deterministic typed confirmation before publishing.
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
  Web App config, Firestore native mode, Cloud Storage snapshot and private campaign-media
  buckets, Vertex AI API enablement, Secret Manager
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
- Configure asynchronous Veo generation with non-secret `VEO_MODEL_ID` (default
  `veo-3.1-generate-001`), `VEO_LOCATION` (default `us-central1`), and
  `CAMPAIGN_MEDIA_BUCKET`. Veo 3.1 processing is US-based even though the Function and private
  media bucket remain in `europe-west2`; expose that boundary to administrators and do not claim
  UK-resident generation. Explicitly provision the managed Vertex AI service identity before
  granting its service-agent role and bucket-scoped object access, rather than relying on lazy
  first-request provisioning. Grant the Function runtime `roles/aiplatform.user` and its required
  bucket-scoped object access. Expire only `work/` media after the configured interval; retain and
  version approved `masters/`.
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
- GitHub Actions should use the current Node.js Active LTS from `.nvmrc`, Go 1.26.6 / `go126` for
  Cloud Run functions, and current runtime-compatible action majors. Deploy Cloud Function runtime
  environment variables from an env vars file rather than comma-separated `--set-env-vars`,
  because values such as `ALLOWED_ORIGINS` contain commas.
- Pin third-party Actions to immutable commit SHAs and serialize both staging and production
  deployment jobs without cancellation. Deployment smoke tests must require the current
  public-statistics schema so outdated snapshots regenerate under the Function runtime
  identity, without giving the GitHub deployer direct member-data access.
- Keep runtime, provider, dependency and action versions current. Use the latest production
  Active LTS for Node, latest GCP-supported Go runtime, current stable OpenTofu, latest
  compatible provider major, and latest compatible stable package releases. Commit lockfiles,
  run weekly Dependabot checks for npm, Go modules, Actions and OpenTofu, and require migration
  guide review plus full tests/build/provider validation for major updates.
- All PRs have a read-only validation job. Only PRs authored by the repository owner whose
  head branch belongs to this repository may automatically request OIDC, staging secrets, or
  a staging deployment. All other PRs stop after validation, and external workflows require
  maintainer approval before that validation runs.
- Repository-owner same-repository PRs deploy to staging preview channels and run smoke tests directly
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
- Custom magic-link, Join re-engagement and member-referral messages share one responsive
  inline-styled HTML shell with the established text masthead, hero image and pill-shaped action
  buttons. Do not insert the site logo image into transactional or campaign email templates.
  Campaign prose is maintained as embedded `email-templates/*.md.tmpl` Go templates and rendered
  into both escaped HTML and plain text; mandatory consent/unsubscribe footers remain structural
  composition data rather than editable campaign Markdown.
- Admin campaign preview responses return that exact generated HTML with placeholder-only private
  links. The browser renders it in a sandboxed `srcdoc` iframe and keeps the plain-text alternative
  available in a disclosure, without exposing recipient data.
- The admin campaign workspace also supports reusable custom member campaigns. Store each
  immutable run as an `emailCampaigns` parent document with its name, subject, Markdown,
  source campaign ID, status, eligible/sent/failed/remaining totals, batch count and timestamps;
  keep hashed per-address delivery documents in its `deliveries` subcollection. History must
  recover older delivery-only campaign paths, return no addresses, and clone rather than mutate a
  campaign that has started sending. Present history before a single campaign-tools panel and
  provide a direct create-new shortcut. Use a compact page-header navigator and accessible tabs
  for registration reminders, member referrals and freeform campaigns; keyboard arrow/Home/End
  navigation and history actions must select the relevant panel. Keep safety guidance next to the
  preview/send confirmation it explains rather than using a visually dominant page-level warning.
  Drafts may be reopened, and partial custom runs may continue only after re-previewing the exact
  unchanged saved content; editing a run with deliveries must create a new sourced run.
- Custom campaigns target the canonical-email intersection of contact-consenting Join records
  and verified Firebase accounts. Available literal `{{name}}` substitutions are
  `membersJoined`, `membersVerified`, `memberFirstName`, `memberLastName`, `memberTittle`
  (plus corrected alias `memberTitle`), `memberJoined`, `memberVerified`, `memberVehicles`,
  `vehiclesRegisteredCount`, and `vehiclesSoHReadingsCount`. Reject other template actions.
  `memberVehicles` is JSON containing only the member's non-VIN vehicle fields and per-vehicle
  SoH reading count. Persist the first observed verified timestamp on the private member record;
  legacy values backfilled from Firebase Auth metadata are marked inferred.
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
- `20-instagram-campaign-publishing.md`
- `21-clean-room-reconstruction-contract.md`
