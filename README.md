# I-PACE Owners' Advocacy Group

An independent advocacy group for Jaguar I-PACE owners affected by traction battery faults,
H441/H448 and H57x recall or customer-notice work, battery degradation, and warranty
uncertainty.

**Website:** [ipace-owners.org](https://ipace-owners.org) (live since 17 July 2026)

---

## Project overview

Static, mobile-first, accessible website built with:

- **[Eleventy 3](https://www.11ty.dev/)** — static site generator
- **Nunjucks + Markdown** — templating and content
- **Custom CSS** — mobile-first design system, no Tailwind or Bootstrap
- **Vanilla JavaScript** — mobile nav, Firebase Authentication, multi-step forms
- **Firebase Authentication** — passwordless email-link member authentication
- **Cloud Functions for Firebase, written in Go** — form handling and member/admin APIs
- **Cloud Firestore** — canonical structured data store
- **Cloud Storage** — generated member/public JSON snapshots and future evidence blobs
- **Firebase Hosting / Google Cloud** — staging and production hosting
- **OpenTofu/Terraform** — GCP resource configuration

---

## Local setup

### Prerequisites

- Node.js 24 LTS
- npm 9+
- Go 1.26+
- Make
- Firebase CLI, installed through `npm install`
- OpenTofu 1.12.3+ for infrastructure changes

Most local and CI tasks are exposed through `make`. Run this to list available targets:

```bash
make
```

### Install

```bash
nvm use
make install
```

Node-backed Make targets verify that the active Node major matches `.nvmrc` and stop with a
clear error if another runtime is selected. GitHub Actions also reads `.nvmrc`, keeping local,
CI and deployment work on Node 24 LTS.

### Development server

```bash
make dev
```

Starts the Eleventy development server. Test Firebase Authentication and Go Functions with
the Firebase emulators or against the staging Firebase project. Build-time Firebase web
config is read from environment variables:

```bash
FIREBASE_WEB_API_KEY=...
FIREBASE_AUTH_DOMAIN=...
FIREBASE_PROJECT_ID=...
FIREBASE_APP_ID=...
FIREBASE_STORAGE_BUCKET=...
```

Do not commit `.env` files containing real values.

### Launch and full presentation modes

The public site defaults to the recruitment-focused `launch` presentation configured by
`site.defaultMode`. Append `?site-mode=full` to any page to enable the complete evidence and
data-oriented presentation for the current browser session. Append `?site-mode=launch` to
return to launch mode. This flag controls public discoverability only; Firebase authentication
and server-side authorization continue to protect member and admin data.

### Version policy

Use the latest stable, supported version that is appropriate for production, not simply the
highest preview or Current release number:

- Node.js uses the latest Active LTS major in `.nvmrc` and `package.json`. Node 24 is the
  production line; Node 26 remains a Current release until it reaches LTS.
- Go uses the latest runtime supported by second-generation Google Cloud Functions. The
  module and deployments currently use Go 1.26 / `go126`.
- OpenTofu uses the current stable release line. Provider constraints follow the latest
  supported major, while `.terraform.lock.hcl` pins the exact reviewed provider versions.
- npm and Go dependencies use current compatible stable releases recorded in their lock or
  checksum files. GitHub Actions use their latest supported major releases.

Dependabot checks npm, Go modules, GitHub Actions and OpenTofu providers weekly, grouping
compatible minor and patch updates by ecosystem to reduce PR noise. Update PRs
must pass `make lint`, `make test`, `make build`, and
`tofu -chdir=infra/opentofu/env validate`; major
updates require release-note and migration-guide review before merging. Do not retain an old
major solely to avoid addressing a documented migration.

### Linting

```bash
make lint
```

Checks JavaScript, CSS, Markdown, JSON/YAML, Nunjucks templates, Bash, Go, OpenTofu/HCL,
and SVG/XML. Each check is also available through the corresponding `make lint-*` target.

### Security audit

```bash
make audit
```

Runs `npm audit` with a high-severity failure threshold and pinned `govulncheck` analysis for
reachable Go vulnerabilities. GitHub Actions additionally runs CodeQL and dependency review
on pull requests and a weekly schedule. Firebase PR previews receive a blocking passive OWASP
ZAP baseline scan after deployment smoke tests pass.

### Production build

```bash
make build
```

Output is written to `_site/`.

### Tests

```bash
make test
```

Runs the Node test suite for form wiring, auth UI behaviour, deployment configuration, and
the Go Cloud Function tests. They can also be run separately:

```bash
make test-node
make test-go
```

### Clean

```bash
make clean
```

Removes the `_site/` directory.

---

## Project structure

```
src/
  *.md / *.njk     # Top-level page templates
  member/          # Authenticated account, vehicle and evidence workspace pages
  admin/           # Admin-gated review and human-controlled outreach tools
  updates/         # Update/news posts (.md)
  _data/           # Global data (site.json, navigation.json)
  _includes/
    layouts/       # Page layouts (base, page, form-page)
    partials/      # Shared UI, SEO, auth-gate and Join-result partials
  assets/
    css/site.css   # Main stylesheet
    js/
      main.js            # Mobile menu, nav helpers
      identity.js        # Firebase Auth email-link integration
      member-auth.js     # Server-verified member/admin data loading
      member-dashboard.js # Vehicle tabs, SoH graph and service/fault editing
      multistep-form.js  # Multi-step form controller
      outreach-assistant.js # Facebook search-link and reply helper; no automation
      public-stats.js    # Public aggregate-statistics rendering
      site-mode.js       # Launch/full presentation selection
public/            # Favicons, images and other root-level static assets
functions/firebase-go/ # Go Cloud Functions for Firebase
infra/opentofu/   # GCP/Firebase infrastructure
prompts/           # Sequenced prompts and architecture blueprint
.eleventy.js       # Eleventy configuration
firebase.json      # Firebase Hosting rewrites and security headers
```

### Preserved visual assets

`public/` is copied unchanged to the site root. The approved favicon, hero, logo, QR code,
and original/photographic business-card SVG and PNG pairs are committed under `public/` and
`public/images/`. The SVG card variants are self-contained so embedded hero/QR artwork does
not depend on neighbouring files. There is currently no committed print PDF; generate and
proof one from the approved SVG/PNG sources before describing a PDF as downloadable or
recoverable.

---

## GCP / Firebase deployment

The target deployment is two Firebase/GCP projects:

- **staging** — PR preview channels and pre-merge testing.
- **production** — `ipace-owners.org`, deployed after merge to `main`.

Infrastructure is configured with OpenTofu under `infra/opentofu/`. The reusable module is
in `infra/opentofu/modules/ipace-owners`; the single environment root is
`infra/opentofu/env`. Staging and production use the same root with different tfvars.
Use separate OpenTofu workspaces so the two environments cannot share one local state file.

```bash
make infra-config ENV=staging
make infra-plan ENV=staging
make deploy-hosting-env ENV=staging

make infra-config ENV=production
make infra-plan ENV=production
make deploy-hosting-env ENV=production
```

`deploy-hosting-env` applies all GCP/Firebase infrastructure for the selected environment,
not only the Hosting resource. It requires an explicit `ENV=staging` or `ENV=production`.
By default it uses `infra/opentofu/env/<environment>.tfvars`; override that with
`TFVARS=path/to/file.tfvars` when needed.

The infrastructure targets check the current gcloud user and Application Default
Credentials. If either credential is missing or expired, they start the relevant
`gcloud auth login` flow. They set the ADC quota project when it already exists, initialise
OpenTofu, and select or create the workspace matching `ENV` before planning or applying.
Useful individual targets are `make infra-auth`, `make infra-init`, and
`make infra-workspace`, each with the same `ENV` and optional `TFVARS` arguments.

Do not commit real `*.tfvars` files. Use the checked-in `staging.tfvars.example` and
`production.tfvars.example` files as templates, or provide values with `TF_VAR_*`.

The OpenTofu config can create the GCP project, then enables Firebase, Firestore, Cloud
Functions, Cloud Storage, Secret Manager and GitHub Workload Identity Federation. It also
creates the Firebase Web App, configures Firebase Authentication for passwordless email
sign-in, creates service accounts for GitHub deployment and Functions runtime, and reads
the generated Firebase web config needed by the site build. Each environment uses a named
Firestore database whose ID matches its GCP project ID; Functions receive that value as
`FIRESTORE_DATABASE_ID` and do not use the `(default)` database. The old default database
is abandoned from OpenTofu state during replacement rather than deleted; migrate any data
that must be retained before directing production Functions to the named database. During
initial infrastructure rollout, the Function environment generator derives the database ID
from `FIREBASE_PROJECT_ID` if the new GitHub environment variable has not been populated.

Every pull request runs read-only lint, test, and build validation. GitHub requires a
maintainer to approve workflows from every external contributor. Only pull requests authored
by the repository owner whose head branch belongs to this repository proceed automatically
to the `staging` environment and request deployment credentials; all other pull requests stop
after validation.

PR deployments do not depend on `stage.ipace-owners.org`. The staging workflow first
creates the PR's Firebase Hosting preview channel and adds the generated hostname to
Firebase Auth's authorized domains. The Go backend is deployed as a single `Api` Function;
Firebase Hosting rewrites `/api/**` to that Function, which routes to the in-process
handlers. The workflow deploys `Api` only when backend code, Firebase rewrites, Function
environment generation, Make deploy logic, or deployment workflow files changed. If `Api`
is redeployed, the workflow refreshes the preview channel so Hosting rewrites use the newly
deployed Function revision. Otherwise the existing staging `Api` revision is reused and the
preview still runs smoke tests. Smoke testing requires the current public-statistics schema;
the deployed Function regenerates an outdated snapshot using its runtime identity. The
GitHub deployer is not granted direct member-data access. Private member snapshots remain
write-triggered and self-heal on authenticated first read. Staging deployments are
serialized because they share one
Firebase Auth configuration and one Cloud Functions backend. The allowlist updater removes
stale PR preview entries while retaining permanent authorized domains. OpenTofu grants the
GitHub deployer a custom role containing only `firebaseauth.configs.get` and
`firebaseauth.configs.update`; apply staging infrastructure after changing these permissions
and before rerunning the preview workflow.

Firebase does not permit preview or default `web.app` domains as Identity Toolkit's
`linkDomain`. Preview emails therefore use Firebase's default action-handler domain and the
validated PR preview request origin as `continueUrl`; production continues to use
`ipace-owners.org` as its custom action-link domain. Preview origins are accepted only when
they match the current Firebase staging project preview-host pattern, not any arbitrary
`web.app` host.

The same OpenTofu module bootstraps the GitHub Actions `staging` and `production`
environments. It creates the environment variables and secrets consumed by the deploy
workflows:

- `GCP_WORKLOAD_IDENTITY_PROVIDER_STAGING` / `GCP_WORKLOAD_IDENTITY_PROVIDER_PRODUCTION`
- `GCP_DEPLOYER_SERVICE_ACCOUNT_STAGING` / `GCP_DEPLOYER_SERVICE_ACCOUNT_PRODUCTION`
- `GCP_FUNCTIONS_SERVICE_ACCOUNT_STAGING` / `GCP_FUNCTIONS_SERVICE_ACCOUNT_PRODUCTION`
- `FIREBASE_WEB_API_KEY_STAGING` / `FIREBASE_WEB_API_KEY_PRODUCTION`
- `VIN_PEPPER_STAGING` / `VIN_PEPPER_PRODUCTION`
- `FIREBASE_STAGING_PROJECT_ID` / `FIREBASE_PRODUCTION_PROJECT_ID`
- `FIRESTORE_DATABASE_ID_STAGING` / `FIRESTORE_DATABASE_ID_PRODUCTION`
- `FIREBASE_AUTH_DOMAIN_*`, `FIREBASE_APP_ID_*`, `FIREBASE_STORAGE_BUCKET_*`
- `SNAPSHOT_BUCKET_*`, `ALLOWED_ORIGINS_*`, `FIREBASE_EMAIL_CONTINUE_URL_*`
- `FIREBASE_EMAIL_LINK_DOMAIN_*`, which makes Firebase Auth action links use the
  environment's verified Firebase Hosting custom domain

Bootstrap requirements:

- Set `GITHUB_TOKEN` locally with permission to administer repository Actions
  environments, variables and secrets.
- For an existing GCP project, provide `project_id`.
- If `create_gcp_project = true`, either provide `project_id` explicitly or leave it empty
  and let OpenTofu derive `${project_id_prefix}-${environment}`. The default prefix creates
  `ipace-owners-staging` or `ipace-owners-production`; override `project_id` if that global
  ID is unavailable. Also provide either `gcp_org_id` or `gcp_folder_id`, plus
  `billing_account`.
- The Google credentials running OpenTofu need enough bootstrap permission to create or
  manage the target project, enable services, create service accounts, create Workload
  Identity pools, create Secret Manager secrets and write project IAM bindings.
- If using local Application Default Credentials, Firebase APIs require a quota project.
  The provider sends `quota_project` as `x-goog-user-project`; by default this is the
  environment project, or you can set `quota_project` in tfvars to an existing bootstrap
  project. You can also align local ADC with:
  `gcloud auth application-default set-quota-project ipace-owners-staging`.
- Provide `site_url`, used as the Firebase email-link continue URL for that environment.
  OpenTofu adds the host from `site_url` to Firebase Auth authorized domains. Add any
  extra hosts, such as `www.ipace-owners.org`, with `firebase_auth_authorized_domains`.
  Add the required DNS records and wait for the Firebase Hosting custom domain to become active
  before deploying Functions with that host as `FIREBASE_EMAIL_LINK_DOMAIN`.
- Firebase web API keys identify the Firebase project and are expected to appear in web
  authentication action URLs. Restrict each generated key to the Firebase APIs the site
  uses in Google Cloud API Credentials; do not handle it as a server credential. The
  one-time action code in an email link is sensitive and must never be logged.
- Provide `vin_pepper` through an uncommitted tfvars file or `TF_VAR_vin_pepper`.
- Set `manage_github_actions = false` only if you want to create the GCP resources without
  touching GitHub repository settings.

Firebase API keys, Firebase app IDs, auth domains, storage bucket names, Workload Identity
provider names and service account emails are generated by OpenTofu and written into the
GitHub Actions environment settings automatically.

### Firebase authentication emails

OpenTofu stores future account-management email designs in
`infra/opentofu/modules/ipace-owners/templates/auth-email/`, but does not PATCH those templates
while the product uses passwordless email-link sign-in. Identity Platform rejects the unrelated
password-reset and verification templates with `EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED`; email-link
sign-in uses Firebase's fixed default template. The reconciliation script manages the supported
locale/delivery settings and custom sender-domain verification. Do not PATCH
`notification.sendEmail.callbackUri` for Firebase's default email provider; Identity Platform
rejects that field with `EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED`. Passwordless action and continue
domains are supplied per request by the Function's `linkDomain` and `continueUrl` settings.

Set `firebase_auth_email_domain` to a dedicated automated-mail subdomain:
`auth.stage.ipace-owners.org` for staging and `auth.ipace-owners.org` for production. This
keeps Firebase's sender authentication separate from human mail at `@ipace-owners.org` and
avoids SPF collisions at the apex. It does not create a mailbox and does not require Google
Workspace, a registrar transfer, a nameserver change, or an MX-record change.

The first apply starts Firebase domain verification. In the matching Firebase project, open
Authentication > Templates and copy every TXT and CNAME record shown for the sender domain
into Fasthosts DNS. Record names may be displayed as fully qualified names; enter them using
Fasthosts' expected relative-host format. Do not edit or remove the apex MX records used by
Fasthosts webmail. Allow DNS to propagate, then run:

```bash
make infra-email-domain ENV=production
```

This command safely reconciles the supported email settings and activates the sender domain
once Firebase reports verification success. Keep a single SPF TXT record at each selected
sender subdomain; merge includes if another sender is ever added instead of publishing a
second SPF record. Normal inbound mail continues to follow the apex MX records to Fasthosts.
Template updates and domain activation use separate API calls: never include
`notification.sendEmail.dnsInfo.useCustomDomain` in the template PATCH before verification,
because Identity Platform rejects the entire update with `EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED`.

Firebase's built-in passwordless email-link sign-in body is not one of the configurable
Identity Platform templates. The custom sender domain and Hosting action-link domain still
apply, but fully branded magic-link copy requires generating the action link server-side and
sending it through a separately selected transactional email provider.

### Join re-engagement campaign

`functions/firebase-go/cmd/reengagement` is an operator-only command for reminding people who
submitted Join but have not completed Firebase passwordless sign-in. It extracts the selected
environment's `joinSubmissions` and Firebase Auth users directly, deduplicates Join submissions by
base address after removing any `+tag`,
and excludes registrations matched by exact email, by email after removing a `+tag`, or by a
case- and punctuation-insensitive display name. Dry run is the default and does not generate live
links or contact Resend:

```bash
make join-reengagement \
  ENV=production \
  RESULTS=/private/tmp/join-reengagement-production-dry-run.csv
```

Use `ARGS='--log-level=debug'` to print each candidate's name, email and submission date. At the
default `info` level, logs contain counts but not personal data. The `0600` results CSV always
contains every extracted Join name, email and submission date, its eligibility status, and the
match reason. A sibling `.manifest.json` file stores the resolved non-secret campaign settings and
counts with the same restricted permissions, making the run reproducible and auditable. Use
`ENV=staging` to run the same
workflow against staging; environment-specific project, database, continue-link and action-link
defaults are selected together.

Review the dry-run results and current eligible count. A live run additionally requires the
Resend sending key in `RESEND_API_KEY`, the configured sender in `RESEND_FROM`, `--send`, an exact
count confirmation, and an interactive typed confirmation. These are the same environment names
used by the deployed app; `RESEND_REPLY_TO` and `RESEND_ASSET_BASE_URL` are read the same way. The
command generates a stable `join-account-verification-<environment>-<UTC date>` campaign ID when
`--campaign-id` is omitted, or accepts an explicit ID for an operator-selected campaign boundary.

The summary reports both raw Auth accounts and canonical Auth identities, exposing duplicate
`+tag`/base-address accounts separately from canonical Auth identities without a corresponding
Join submission. The eligible cohort is the Join set minus only the Join identities actually
matched to Auth by exact email, `+tag` alias, or name.
Auth coverage itself is stricter: names do not satisfy the privacy invariant. Any canonical Auth
email without a canonical Join email makes the command fail after writing the dry-run ledger and
always blocks a live send.

```bash
export RESEND_API_KEY="set-from-your-secure-password-manager"
export RESEND_FROM="I-PACE Owners <members@ipace-owners.org>"
export RESEND_REPLY_TO="contact@ipace-owners.org"
export RESEND_ASSET_BASE_URL="https://ipace-owners.org"

make join-reengagement \
  ENV=production \
  RESULTS=/private/tmp/join-reengagement-production-2026-07.csv \
  ARGS='--send --confirm-count=150 --log-level=debug'
```

Never commit the result CSV or manifest because they contain operational or personal data. The
manifest never contains API keys or other secret values. The command refuses to overwrite either
campaign file, refreshes the complete Auth comparison immediately before confirmation,
rechecks each exact address immediately before sending, creates a fresh time-limited sign-in link
per recipient, uses Resend idempotency keys, paces requests below the default rate limit, and
records delivery status and the Resend message ID incrementally. Re-run dry mode shortly
before any campaign because the eligible count falls as members complete sign-in. Check the Resend
plan's daily quota before sending; a free transactional plan cannot deliver 150 messages in one day.

GitHub Actions automatically deploys repository-owner same-repository PRs to Firebase Hosting preview channels and
deploys `main` to the production Firebase Hosting site. Production deployments are serialized
without cancellation so closely timed merges cannot race Cloud Functions operations.

### SSL and DNS

Firebase Hosting provides managed SSL certificates for connected custom domains. OpenTofu
registers each domain from `firebase_hosting_custom_domains` and outputs its ownership,
hosting and certificate state together with the required DNS changes.

Fasthosts DNS changes are currently manual:

```bash
make deploy-hosting-env ENV=staging
make infra-dns-records ENV=staging

# Add every reported A, TXT, CNAME or other record in Fasthosts Advanced DNS.
# Refresh until ownership, hosting and certificate states become active.
make infra-plan ENV=staging
make infra-dns-records ENV=staging
```

The custom-domain resource deliberately never waits during creation because changing that
provider field later forces destructive replacement. Firebase validates DNS asynchronously;
planning or reading the outputs refreshes its current state without recreating the domain.
Its deletion policy is `PREVENT`, so an accidental replacement or removal fails before the
domain association is removed from state or Firebase.

Repeat for production only after staging is connected. The output combines Firebase Hosting
traffic/ownership records with certificate ACME TXT records. The recommended domains are
`stage.ipace-owners.org` for an optional permanent staging site, `ipace-owners.org` as the
production canonical domain, and `www.ipace-owners.org` as a redirect to the apex. PRs use
their generated Firebase Hosting preview URLs instead. Do not remove or alter existing MX,
SPF, DKIM or DMARC records.

No documented Fasthosts DNS API, RFC 2136 endpoint, or maintained OpenTofu provider was
found, so the configuration deliberately does not automate changes in Fasthosts. Moving
authoritative DNS to Cloud DNS would make the records fully manageable in OpenTofu, but is
not required for Firebase Hosting and would require a careful migration of every DNS record.

### Submission storage

Cloud Firestore is the intended canonical source for structured owner data.

Join submissions, vehicle basics, SoH updates, and service history are handled by routes
behind the single Go `Api` Cloud Function:

- `submit-join` stores membership expressions of interest and consent choices, then sends
  the Firebase passwordless activation link for logged-out users.
- `send-magic-link` is a login-only path for already registered members. It checks for a
  matching Join submission before invoking the configured Firebase-default or
  Admin-SDK/Resend delivery path, and returns a generic response so registration state is
  not exposed to the browser.
- `submit-vehicle-basics` stores the first vehicle registration slice for signed-in users:
  VIN HMAC / final six characters, registration, country, model year, ownership dates,
  mileage, State of Health, measurement date, measurement mileage, and SoH source.
- `submit-soh` appends a dated State of Health reading to a vehicle after verifying that
  the signed-in member owns the referenced record. Earlier readings are retained for
  degradation analysis.
- `upsert-service-event` adds or edits an owned vehicle's dated service, fault, repair,
  recall, or inspection record after server-side ownership verification.
- `member-data` returns only the authenticated member's generated private snapshot after
  Firebase ID-token verification.
- `admin-data` returns Join and vehicle review records only when the Firebase token carries
  an accepted admin claim.
- `public-stats` serves a cacheable aggregate snapshot. Its registered-member headline is
  refreshed from the complete paginated Firebase Auth user list; vehicle and SoH aggregates
  remain consent-filtered and exclude records marked out of public reporting.

Member/account JSON snapshots are regenerated after signed-in Join, vehicle, SoH, and
service-event changes, written to Firestore and optionally Cloud Storage, then served only
through `member-data` after server-side Firebase ID-token verification. If a logged-out Join
has no UID yet, `member-data` builds the snapshot after activation when first needed. Vehicle
and SoH writes also regenerate the anonymised public evidence snapshot in Cloud Storage.

Members may register more than one I-PACE. The member dashboard uses vehicle tabs and shows
one selected car's SoH graph and service/fault timeline at a time.

Set `VIN_PEPPER` as a GCP Secret Manager value and Function environment variable before
collecting VINs. Full VINs are not stored; the Function uses `VIN_PEPPER` to create an HMAC
for deduplication.

### Magic-link delivery troubleshooting

A successful Identity Toolkit response means Firebase accepted the email-link request; it
does not provide mailbox-delivery confirmation. Function logs include a one-way email hash,
masked address, continue host, response size, and whether Firebase echoed the expected email.
They never include the raw address or sign-in link.

If a link does not arrive:

1. Check spam/junk filtering and test a mailbox at a different provider.
2. Confirm the Firebase project is on the intended billing plan and has not reached its
   [Authentication email sending limit](https://firebase.google.com/docs/auth/limits).
   Firebase currently limits email-link sign-in delivery to 5 emails/day on Spark and
   25,000 emails/day on Blaze.
3. Check the Firebase Authentication email template and sender settings.
4. For delivery control, enable the implemented Firebase Admin SDK plus Resend path with the
   documented `RESEND_*` runtime configuration. See
   [Firebase's custom email action link guidance](https://firebase.google.com/docs/auth/admin/email-action-links).

Submitting Join more than once with the same email remains enumeration-safe. Each submission
is retained for review, while Function logs record the previous submission count for that
email hash.

### Admin role assignment

OpenTofu manages the authoritative Firebase administrator set. The shared module always
includes `dan@kanzi.co.uk`; additional administrators are labels mapped to email addresses in
the environment configuration:

```hcl
manage_firebase_admins = true
firebase_admin_users = {
  another_admin = "person@example.org"
}
```

The Google provider has no Firebase Auth user data source. During apply, the tested
`scripts/reconcile-firebase-admins.mjs` bridge lists Identity Platform accounts, resolves each
configured email to its environment-specific UID, preserves unrelated custom claims, grants
`admin: true` to the desired set, and removes only admin access from users no longer listed.
The apply fails rather than silently continuing if a configured account has not completed
Firebase sign-in. Staging and production are reconciled independently. After a claim change,
sign out and request a new magic link so the next ID token contains the current claim.
Signed-in administrators receive an **Admin** header action and an **Admin tools** mobile action
linking to `/admin/outreach/`; these navigation hints use token claims, while the destination
continues to enforce administrator access through the server-verified `admin-data` API.

Disabling `manage_firebase_admins` stops reconciliation but does not revoke existing claims.
Remove unwanted administrators from the map and apply before disabling management.

---

## Known limitations

The following features are **not yet implemented** in this version:

- **Full vehicle/evidence submission persistence** — Join submissions, vehicle basics, SoH
  readings, and member service/fault timeline records are structured slices. Detailed recall,
  battery-work, loan car, payment, responsibility, consent-review, and evidence upload fields
  are not yet stored in the GCP model.
- **Evidence document uploads** — A placeholder message explains what will be supported.
  Requires Cloud Storage for files plus Firestore metadata and Functions integration.
- **Admin review workflow** — The review queue can read server-side data for admins, but
  review status updates, exports, and moderation actions are not yet implemented.
- **Privacy policy** — The current policy is a placeholder. A formal policy is required
  before broader live evidence collection.
- **Later evidence dashboard metrics** — Registered-car and SoH figures are live aggregates.
  Recall, repair, loan-car, warranty, and payment metrics remain unavailable until their
  corresponding form slices are implemented.

---

## Development notes

### CSS

All styles are in `src/assets/css/site.css`. The file is structured as:
reset → design tokens → typography → layout utilities → components → print.

Do not add Tailwind, Bootstrap, or other CSS frameworks.

### JavaScript

Plain vanilla JavaScript, no bundler. The current modules are:

- `main.js` — mobile menu toggle, current-page nav highlighting
- `identity.js` — Firebase Auth email-link and UI state
- `member-auth.js` — authenticated member/admin data loading and account rendering
- `member-dashboard.js` — vehicle tabs, SoH history and service/fault editing
- `multistep-form.js` — generic multi-step form (data-attribute driven)
- `outreach-assistant.js` — admin-only Facebook search-link and editable reply helper; no retrieval or posting automation
- `public-stats.js` — homepage and evidence-dashboard aggregate rendering
- `site-mode.js` — launch/full presentation selection

### Adding pages

Add `.md` or `.njk` files to `src/`. Use `layout: page.njk` for standard pages
or `layout: form-page.njk` for form pages (adds `multistep-form.js`).

### Adding updates

Add `.md` files to `src/updates/` with `title`, `date` and `summary` front matter.
Posts appear automatically on `/updates/`.

### Architecture

See `prompts/09-architecture-overview.md` for the intended architecture using Firebase
Auth, Go Cloud Functions, Firestore for structured data, Cloud Storage for files, and
generated JSON snapshots for read-heavy views.

### Copilot PR reviews

Automatic GitHub Copilot code review is enabled with a repository branch ruleset named
`Automatic Copilot code review`. It targets the default branch, requests Copilot review on
pull requests, and reviews new pushes to existing pull requests.

This is configured in GitHub repository settings under **Rules -> Rulesets**. No repository
workflow or external AI provider API key is required.

### Prompt maintenance

Product-generation prompts live in `prompts/`. Every prompt filename must match
`^\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$` (`xx-name.md`) so the project can be rebuilt or
extended in a controlled order:

- `00-original-project-prompt.md` preserves the initial generation prompt.
- The remaining numbered prompts split the product into foundation, design, content,
  identity, forms, evidence dashboard, backend security/storage, architecture, Function
  components, data modelling, member tooling, operations, stakeholder feedback, launch
  readiness, and reconstruction requirements.
- `20-clean-room-reconstruction-contract.md` is the final route/API/schema/configuration and
  acceptance contract for rebuilding the product from scratch.

When adding or refining prompts, keep the numeric prefix and make the prompt independently
usable. Keep cross-layer implementation details canonical in
`prompts/09-architecture-overview.md` and exact reconstruction inventories canonical in
`prompts/20-clean-room-reconstruction-contract.md`; feature prompts may repeat only the
details needed to apply them safely.

Keep prompts in sync with implemented behaviour so the project can be recreated from
the prompt set and README alone.

### Reconstruction and documentation checks

The Node test suite includes reconstruction-contract checks that compare the maintained
prompts and README with the route/API inventory, browser JavaScript modules, Firestore
collection names, runtime configuration, and preservation-critical visual assets. These
checks catch structural drift; they do not prove that an agent can reproduce the product
from prose alone.

For a genuine reproducibility test, follow the isolated clean-room procedure in
`prompts/20-clean-room-reconstruction-contract.md`: provide only AGENTS.md, all numbered
prompt files, approved public assets, and separately secured configuration in a new
repository, then run the full acceptance checklist and record every manual intervention.

---

## Contributing

This site is maintained by volunteers. If you have skills in web development, data, legal,
or consumer rights and want to help, please [contact us](/contact/) or indicate your interest
when joining the group.

### Quality and testing requirements

- Tests are required for behavioural changes (Functions, Firebase Auth handoff, form submission wiring,
  shared utilities).
- Run `make test` for behavioural changes and ensure all tests pass.
- Run `make build` for every change and ensure the site builds cleanly.
- Pressure-test your changes locally (happy path, error paths, and access-control paths where relevant)
  before opening a PR.

### Pull requests and code review

- All changes must be submitted via pull requests.
- Every PR should include:
  - What changed and why.
  - Which files were modified.
  - How to verify locally.
  - Which tests were added/updated (or why tests were not needed).
- Use Copilot automatic review as a first pass, but require human review before merge.
- Do not merge until `make build` and `make test` pass.

### Commit message conventions

Use semantic commit messages in this format:

`type(scope): description`

Common types:
- `feat` new features or pages
- `fix` bug fixes and validation corrections
- `test` new or updated tests
- `refactor` restructuring without behaviour change
- `docs` documentation and prompt updates
- `style` non-behavioural styling changes
- `chore` dependencies and housekeeping

---

## Licence

Content and code are copyright the I-PACE Owners' Advocacy Group contributors. Manufacturer
and vehicle names are used descriptively; the site does not use Jaguar/JLR logos or badges as
group branding. Committed vehicle artwork is original or generated for this project.
