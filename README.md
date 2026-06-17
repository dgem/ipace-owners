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

- Node.js 20+
- npm 9+
- Go 1.23+
- Firebase CLI, installed through `npm install`
- OpenTofu 1.8+ for infrastructure changes

### Install

```bash
npm install
```

### Development server

```bash
npm run dev
```

Starts the current local development server. During the GCP migration, the static site can
also be run directly with:

```bash
npm run dev:eleventy
```

To test Firebase Authentication and Go Functions locally, run the Firebase emulators or test
against the staging Firebase project. Build-time Firebase web config is read from
environment variables:

```bash
FIREBASE_WEB_API_KEY=...
FIREBASE_AUTH_DOMAIN=...
FIREBASE_PROJECT_ID=...
FIREBASE_APP_ID=...
FIREBASE_STORAGE_BUCKET=...
```

Do not commit `.env` files containing real values.

### Production build

```bash
npm run build
```

Output is written to `_site/`.

### Tests

```bash
npm test
```

Runs the Node test suite for form wiring, auth UI behaviour, and legacy backend guards.
Go Cloud Functions are tested separately with:

```bash
cd functions/firebase-go
go test ./...
```

### Clean

```bash
npm run clean
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

```bash
cd infra/opentofu/env
tofu init
tofu plan -var-file=staging.tfvars
tofu apply -var-file=staging.tfvars

tofu plan -var-file=production.tfvars
tofu apply -var-file=production.tfvars
```

Do not commit real `*.tfvars` files. Use the checked-in `staging.tfvars.example` and
`production.tfvars.example` files as templates, or provide values with `TF_VAR_*`.

The OpenTofu config can create the GCP project, then enables Firebase, Firestore, Cloud
Functions, Cloud Storage, Secret Manager and GitHub Workload Identity Federation. It also
creates the Firebase Web App, service accounts for GitHub deployment and Functions runtime,
and reads the generated Firebase web config needed by the site build.

The same OpenTofu module bootstraps the GitHub Actions `staging` and `production`
environments. It creates the environment variables and secrets consumed by the deploy
workflows:

- `GCP_WORKLOAD_IDENTITY_PROVIDER_STAGING` / `GCP_WORKLOAD_IDENTITY_PROVIDER_PRODUCTION`
- `GCP_DEPLOYER_SERVICE_ACCOUNT_STAGING` / `GCP_DEPLOYER_SERVICE_ACCOUNT_PRODUCTION`
- `GCP_FUNCTIONS_SERVICE_ACCOUNT_STAGING` / `GCP_FUNCTIONS_SERVICE_ACCOUNT_PRODUCTION`
- `FIREBASE_WEB_API_KEY_STAGING` / `FIREBASE_WEB_API_KEY_PRODUCTION`
- `VIN_PEPPER_STAGING` / `VIN_PEPPER_PRODUCTION`
- `FIREBASE_STAGING_PROJECT_ID` / `FIREBASE_PRODUCTION_PROJECT_ID`
- `FIREBASE_AUTH_DOMAIN_*`, `FIREBASE_APP_ID_*`, `FIREBASE_STORAGE_BUCKET_*`
- `SNAPSHOT_BUCKET_*`, `ALLOWED_ORIGINS_*`, `FIREBASE_EMAIL_CONTINUE_URL_*`

Bootstrap requirements:

- Set `GITHUB_TOKEN` locally with permission to administer repository Actions
  environments, variables and secrets.
- For an existing GCP project, provide `project_id`.
- If `create_gcp_project = true`, either provide `project_id` explicitly or leave it empty
  and let OpenTofu derive `${project_id_prefix}-${environment}`. The default prefix creates
  `ipace-owners-staging` or `ipace-owners-production`; override `project_id` if that global
  ID is unavailable. Also provide either `gcp_org_id` or `gcp_folder_id`, plus
  `billing_account`.
- Provide `site_url`, used as the Firebase email-link continue URL for that environment.
- Provide `vin_pepper` through an uncommitted tfvars file or `TF_VAR_vin_pepper`.
- Set `manage_github_actions = false` only if you want to create the GCP resources without
  touching GitHub repository settings.

Firebase API keys, Firebase app IDs, auth domains, storage bucket names, Workload Identity
provider names and service account emails are generated by OpenTofu and written into the
GitHub Actions environment settings automatically.

GitHub Actions deploys PRs to Firebase Hosting preview channels and deploys `main` to the
production Firebase Hosting site.

### SSL and DNS

Firebase Hosting provides managed SSL certificates for connected custom domains. For
`ipace-owners.org`, add the domain in Firebase Hosting, then update DNS records in
Fasthosts exactly as Firebase instructs. Firebase will verify ownership and provision the
certificate automatically. Keep the production cutover separate from staging validation.

### Submission storage

Cloud Firestore is the intended canonical source for structured owner data.

Join submissions and vehicle basics are handled by Go Cloud Functions:

- `submit-join` stores membership expressions of interest and consent choices, then sends
  the Identity magic link for logged-out users.
- `submit-vehicle-basics` stores the first vehicle registration slice for signed-in users:
  VIN HMAC / final six characters, registration, country, model year, ownership dates,
  mileage, State of Health, measurement date, measurement mileage, and SoH source.

Member/account JSON snapshots are regenerated after signup and vehicle changes, written to
Firestore and optionally Cloud Storage, then served only through `member-data` after
server-side Firebase ID-token verification. Public evidence dashboard JSON should be
generated only from anonymised aggregate data.

Members may register more than one I-PACE. The account and member dashboard UX should treat
vehicle records as a list, not as a single profile.

Set `VIN_PEPPER` as a GCP Secret Manager value and Function environment variable before
collecting VINs. Full VINs are not stored; the Function uses `VIN_PEPPER` to create an HMAC
for deduplication.

### Admin role assignment

To grant a member admin access:

Set a Firebase Auth custom claim for the user, for example `admin: true` or
`roles: ["admin"]`. Admin APIs verify this claim server-side.

---

## Known limitations

The following features are **not yet implemented** in this version:

- **Full vehicle/evidence submission persistence** — Join submissions and vehicle basics are
  the first structured slices. Recall, repair, loan car, payment, responsibility,
  consent-review, and evidence upload details are not yet stored in the GCP model.
- **Evidence document uploads** — A placeholder message explains what will be supported.
  Requires Cloud Storage for files plus Firestore metadata and Functions integration.
- **Admin review workflow** — The review queue can read server-side data for admins, but
  review status updates, exports, and moderation actions are not yet implemented.
- **Privacy policy** — The current policy is a placeholder. A formal policy is required
  before broader live evidence collection.
- **Evidence dashboard data** — All figures are illustrative. Real data collection has not begun.

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
- Run `npm test` for behavioural changes and ensure all tests pass.
- Run `npm run build` for every change and ensure the site builds cleanly.
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
- Do not merge until both `npm run build` and `npm test` pass.

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
