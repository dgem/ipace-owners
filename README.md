# I-PACE Owners' Advocacy Group

An independent advocacy group for Jaguar I-PACE owners affected by traction battery faults,
H570/H571/H572 recalls, battery degradation, and warranty uncertainty.

**Website:** [ipace-owners.org](https://ipace-owners.org) (coming soon)

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
make install
```

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

Dependabot checks npm, Go modules, GitHub Actions and OpenTofu providers weekly. Update PRs
must pass `make test`, `make build`, and `tofu -chdir=infra/opentofu/env validate`; major
updates require release-note and migration-guide review before merging. Do not retain an old
major solely to avoid addressing a documented migration.

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
  member/          # Member-gated placeholder pages
  admin/           # Admin-gated placeholder pages
  updates/         # Update/news posts (.md)
  _data/           # Global data (site.json, navigation.json)
  _includes/
    layouts/       # Page layouts (base, page, form-page)
    partials/      # Shared partials (header, footer, nav, card, callout)
  assets/
    css/site.css   # Main stylesheet
    js/
      main.js            # Mobile menu, nav helpers
      identity.js        # Firebase Auth email-link integration
      member-auth.js     # Server-verified member/admin data loading
      multistep-form.js  # Multi-step form controller
public/images/     # Static images
functions/firebase-go/ # Go Cloud Functions for Firebase
infra/opentofu/   # GCP/Firebase infrastructure
prompts/           # Sequenced prompts and architecture blueprint
.eleventy.js       # Eleventy configuration
firebase.json      # Firebase Hosting rewrites, headers and Functions config
```

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

PR deployments do not depend on `stage.ipace-owners.org`. The staging workflow first
creates the PR's Firebase Hosting preview channel, configures Functions with that generated
`web.app` URL for CORS and passwordless email links, adds the generated hostname to Firebase
Auth's authorized domains, then refreshes the channel so Hosting rewrites use the newly
deployed Function revisions. Staging deployments are serialized because they share one
Firebase Auth configuration and one set of Cloud Functions. The allowlist updater removes
stale PR preview entries while retaining permanent authorized domains. OpenTofu grants the
GitHub deployer a custom role containing only `firebaseauth.configs.get` and
`firebaseauth.configs.update`; apply staging infrastructure after changing these permissions
and before rerunning the preview workflow.

Firebase does not permit preview or default `web.app` domains as Identity Toolkit's
`linkDomain`. Preview emails therefore use Firebase's default action-handler domain and the
PR preview URL as `continueUrl`; production continues to use `ipace-owners.org` as its
custom action-link domain.

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

GitHub Actions deploys PRs to Firebase Hosting preview channels and deploys `main` to the
production Firebase Hosting site.

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

Join submissions, vehicle basics, SoH updates, and service history are handled by Go Cloud Functions:

- `submit-join` stores membership expressions of interest and consent choices, then sends
  the Identity magic link for logged-out users.
- `submit-vehicle-basics` stores the first vehicle registration slice for signed-in users:
  VIN HMAC / final six characters, registration, country, model year, ownership dates,
  mileage, State of Health, measurement date, measurement mileage, and SoH source.
- `submit-soh` appends a dated State of Health reading to a vehicle after verifying that
  the signed-in member owns the referenced record. Earlier readings are retained for
  degradation analysis.
- `upsert-service-event` adds or edits an owned vehicle's dated service, fault, repair,
  recall, or inspection record after server-side ownership verification.
- `public-stats` serves a cacheable, consent-filtered aggregate snapshot containing current
  vehicle counts, SoH totals and distributions, without exposing member records.

Member/account JSON snapshots are regenerated after signup, vehicle, SoH, and service-event changes, written to
Firestore and optionally Cloud Storage, then served only through `member-data` after
server-side Firebase ID-token verification. Vehicle and SoH writes also regenerate the
anonymised public evidence snapshot in Cloud Storage.

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
4. For delivery tracking and control, generate sign-in links with the Firebase Admin SDK
   and send them through a configured transactional email/SMTP provider, as described in
   [Firebase's custom email action link guidance](https://firebase.google.com/docs/auth/admin/email-action-links).

Submitting Join more than once with the same email remains enumeration-safe. Each submission
is retained for review, while Function logs record the previous submission count for that
email hash.

### Admin role assignment

To grant a member admin access:

Set a Firebase Auth custom claim for the user, for example `admin: true` or
`roles: ["admin"]`. Admin APIs verify this claim server-side.

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

Plain vanilla JS, no bundler. Three files:
- `main.js` — mobile menu toggle, current-page nav highlighting
- `identity.js` — Firebase Auth email-link and UI state
- `multistep-form.js` — generic multi-step form (data-attribute driven)

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

Product-generation prompts live in `prompts/`. They are sequenced with two-digit prefixes
so the project can be rebuilt or extended in a controlled order:

- `00-original-project-prompt.md` preserves the initial generation prompt.
- `01-` through `14-` split the product into foundation, design, content, identity, forms,
  evidence dashboard, backend security/storage, architecture, Function components, and
  future evidence/statistics concerns.

When adding or refining prompts, keep the numeric prefix, make the prompt independently
usable, and avoid duplicating live implementation details that belong in README or
`prompts/09-architecture-overview.md`.

Keep prompts in sync with implemented behaviour so the project can be recreated from
the prompt set and README alone.

---

## Contributing

This site is maintained by volunteers. If you have skills in web development, data, legal,
or consumer rights and want to help, please [contact us](/contact/) or indicate your interest
when joining the group.

### Quality and testing requirements

- Tests are required for behavioural changes (Functions, Identity handoff, form submission wiring,
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

Content and code are copyright the I-PACE Owners' Advocacy Group contributors.
No JLR / Jaguar trademarks, logos, or copyrighted imagery are used.
