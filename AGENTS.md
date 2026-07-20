# AGENTS.md — I-PACE Owners' Advocacy Group

## Project overview

This is the I-PACE Owners' Advocacy Group website — a static, mobile-first, accessible
public site built with Eleventy (11ty) and deployed to Firebase/GCP.

## Repository structure

```
.
├── src/
│   ├── *.md / *.njk     # Top-level Markdown and Nunjucks page templates
│   ├── member/          # Member-only placeholder pages
│   ├── admin/           # Admin-only placeholder pages
│   ├── updates/         # Update/news posts (.md)
│   ├── _data/           # Global data files (site.json, navigation.json)
│   ├── _includes/
│   │   ├── layouts/     # Page layouts (base.njk, page.njk, form-page.njk)
│   │   └── partials/    # Reusable partials (header, footer, nav, card, callout)
│   └── assets/
│       ├── css/site.css # Main stylesheet — all styles, no Tailwind
│       └── js/
│           ├── main.js            # Mobile menu, nav current-page detection
│           ├── identity.js        # Firebase Auth email-link integration
│           ├── member-auth.js     # Server-verified member/admin data loading
│           └── multistep-form.js  # Generic multi-step form controller
├── public/images/       # Static images (passed through to _site)
├── functions/firebase-go/ # Go Cloud Functions for Firebase/GCP
├── infra/opentofu/      # GCP/Firebase infrastructure
├── prompts/             # Sequenced prompts for rebuilding and evolving the product
├── .eleventy.js         # Eleventy configuration
├── firebase.json        # Firebase Hosting rewrites, headers and Functions config
├── Makefile             # Shared local/CI command entrypoints
├── package.json         # npm scripts and dependencies
├── AGENTS.md            # This file
└── README.md            # Setup and deployment documentation
```

## Development commands

```bash
make            # List available local/CI targets
make install    # Install dependencies for local development
make dev        # Start the local development server
make build      # Build production site to _site/
make test       # Run Node and Go tests
make clean      # Remove _site/ directory
make functions  # List deployed Cloud Function entrypoints
```

Eleventy v3 is used. The Eleventy config file is `.eleventy.js` (CommonJS format).

## Technology choices

- **Static site generator:** Eleventy 3 (`@11ty/eleventy`)
- **Templates:** Nunjucks (`.njk`) for complex pages; Markdown for content pages
- **CSS:** Custom CSS only — no Tailwind, no Bootstrap, no utility frameworks
- **JavaScript:** Plain vanilla JS — no React, Vue, Svelte, or heavy frameworks
- **Authentication:** Firebase Authentication passwordless email links.
- **Hosting:** Firebase Hosting / Google Cloud
- **Backend:** Go Cloud Functions + Cloud Firestore for structured data; Cloud Storage for generated snapshots and future binary evidence files

## Version policy

- Use the latest stable production-supported toolchain: the current Node.js Active LTS,
  the latest Go runtime supported by GCP Cloud Functions, and the current stable OpenTofu.
- Keep OpenTofu providers on their latest compatible major and commit the exact selections
  in `.terraform.lock.hcl`. Do not retain an old provider major to avoid migration work.
- Keep npm and Go dependencies current through committed lock/checksum files. Dependabot
  checks npm, Go modules, GitHub Actions and OpenTofu providers weekly.
- Review major-version migration guides and run `make test`, `make build`, and
  `tofu -chdir=infra/opentofu/env validate` before merging dependency updates.
- Prefer an Active LTS runtime over a numerically newer non-LTS release for production.

## CSS conventions

All CSS is in `src/assets/css/site.css`. It is structured as:
1. Modern reset
2. Design tokens (CSS custom properties on `:root`)
3. Base typography
4. Layout utilities (`.container`, `.section`, `.stack`, `.cluster`, `.grid`, etc.)
5. Components (header, nav, hero, cards, callouts, buttons, badges, forms, footer)
6. Print styles

**Do not use Tailwind or any utility-first CSS framework.** Use the existing design tokens
and component classes. If you add a new component, add it to the CSS file in the correct
section.

## Design tokens

Defined in `:root` in `site.css`. Key tokens:

| Token | Value | Use |
|---|---|---|
| `--color-primary` | `#12324a` | Navy — header, footer, headings |
| `--color-accent` | `#0f766e` | Teal — CTAs, links, accents |
| `--color-bg` | `#f7f8fb` | Page background |
| `--color-surface` | `#ffffff` | Card/panel backgrounds |
| `--color-text` | `#111827` | Body text |
| `--color-text-muted` | `#4b5563` | Secondary text |
| `--font-sans` | `system-ui, -apple-system, …` | All UI text |

## JavaScript conventions

- All JS is plain ES5/ES6 — no build step, no bundler, no TypeScript.
- Each JS file is an IIFE (`(function() { 'use strict'; … })()`).
- Scripts are loaded with `defer` in the base layout.
- `multistep-form.js` works from data attributes (`data-multistep-form`, `data-step`,
  `data-next`, `data-prev`, `data-submit`, `data-progress`, etc.) and is not hardcoded
  to any specific form.

## Authentication (Firebase Auth)

- Firebase Auth SDK is loaded from the CDN in `base.njk` when build-time config is present.
- `identity.js` completes email-link sign-in, updates the header UI, submits passwordless
  magic-link forms, and injects Firebase ID tokens into protected form submissions.
- On Join form completion, `identity.js` calls `POST /api/submit-join` once. The Go
  Function stores the Join submission and dispatches a Firebase email sign-in link
  server-side.
- Existing registered users can request another magic link through
  `POST /api/send-magic-link` without resubmitting the Join form. The Function checks
  Join submissions first and suppresses email side effects for unregistered addresses
  while still returning a generic response.
- Signed-in vehicle basics are stored by `POST /api/submit-vehicle-basics`.
- Signed-in members append SoH history through `POST /api/submit-soh`; the Function verifies
  vehicle ownership server-side.
- Signed-in members add or edit service/fault timeline records through
  `POST /api/upsert-service-event`; the Function verifies both record and vehicle ownership.
- Member/admin data fetches must send the Firebase ID token in the `Authorization` header.
- Members may register multiple vehicles. Account/member pages should render vehicle lists,
  not a single vehicle assumption.
- Member/account JSON snapshots are private data and must be served only after
  `MemberData` verifies Firebase Auth server-side. Public static JSON is for anonymised
  aggregate data only.
- Member pages use `data-auth-gate` / `data-auth-content` attributes.
- Admin pages use `data-admin-gate` / `data-admin-content` attributes.
- **Frontend gating is not sufficient for real data access.** Go Functions must verify
  Firebase ID tokens server-side.

## Adding a new page

1. Create a `.md` or `.njk` file in `src/`.
2. Add front matter with at least `layout`, `title`, and `description`.
3. For a standard content page, use `layout: page.njk`.
4. For a multi-step form page, use `layout: form-page.njk` and add `multiStepForm: true`.
5. Add navigation links to `src/_data/navigation.json` if needed.

## Prompt maintenance

- Product-generation prompts live in `prompts/`.
- Every prompt file must be prefixed with a two-digit sequence number, starting with
  `00-original-project-prompt.md`.
- Split prompts by product concern, feature, or implementation phase rather than keeping
  one large prompt.
- Preserve historical prompts where useful, then add refined prompts for future work.
- Treat `09-architecture-overview.md` as the canonical architecture reference and keep it
  aligned with README and the component prompts.
- Treat `20-clean-room-reconstruction-contract.md` as the final clean-room acceptance
  contract. Keep its route, API, schema, configuration, asset-preservation, and external
  backup inventories aligned with implementation.
- **Keep prompts in sync with the actual state of the project.** After implementing a
  feature, update the relevant prompt file to reflect what was built so that the project
  can be recreated from the prompts and README alone. If behaviour changes (e.g., a new
  Function is added, a form field is removed), update the corresponding prompt.
- **Externalise durable conversation knowledge into prompts.** When the user clarifies a
  product decision, discovers an operational constraint, diagnoses a deployment/auth/data
  issue, or chooses an approach that future work depends on, update the relevant numbered
  prompt even if the code change is small or already complete. Examples include Firebase
  preview/auth quirks, CI smoke-test behaviour, data-publication rules, navigation/UX
  decisions, provider/runtime version policy, and manual DNS/deployment runbook steps.
- Before finishing any non-trivial task, do a prompt-drift check: identify whether the
  change affects architecture, auth, forms, data storage, CI/deploy, UX/content, or
  operations. If yes, update the matching prompt file; if no prompt fits, add or split a
  sequenced prompt rather than leaving the knowledge only in chat history.

## Testing

- **Tests are required for all behavioural changes.** Any change to Go Functions, form
  submission wiring, Firebase Auth handoff, or shared utilities must include or update
  tests.
- Run `make test` before considering a change complete. All existing tests must pass.
- Tests should cover:
  - Server-side validation and authorization paths (unauthenticated, authenticated, admin).
  - Input sanitisation and edge cases (empty bodies, invalid JSON, honeypot fields).
  - Storage-shaping logic (Firestore document structure, generated JSON snapshots, metadata, HMAC behaviour).
  - Magic-link handoff behaviour (new vs existing user flow).
- Content-only changes (Markdown copy, CSS, static templates) need not add tests but must
  pass `make build`.

## Code review and pull requests

- **All changes must be submitted via pull requests.** Do not push directly to the default
  branch.
- Before pushing additional commits to an existing PR branch, cancel any in-progress staging
  deployment for that same branch. Do not cancel deployments for other PR branches; staging
  deployments remain globally serialized because they share infrastructure.
- Every PR must include a clear description covering:
  - What changed and why.
  - Which files were added or modified.
  - How to verify the change locally (e.g., `make dev` and navigate to X).
  - Whether tests were added or updated.
- Whenever reporting that a PR was created, updated, pushed to, or is ready for review,
  include the full clickable GitHub PR URL, not only the PR number.
- **Code review is required before merging.** Use GitHub's automatic Copilot code review
  (configured via repository branch ruleset) as a first pass, but every PR must receive
  human review for logic, security, accessibility, and tone.
- Do not merge until `make build` and `make test` pass cleanly.

## Commit message conventions

Use semantic commit messages in the format `type(scope): description`.

| Type | Use for |
|---|---|
| `feat` | New features or pages |
| `fix` | Bug fixes or validation corrections |
| `test` | Adding or updating tests |
| `refactor` | Code restructuring without behavioural change |
| `docs` | README, AGENTS.md, prompts, architecture docs |
| `style` | CSS changes that don't affect behaviour |
| `chore` | Dependency updates, config changes, housekeeping |

Examples:
- `feat(forms): add vehicle evidence upload placeholder`
- `fix(functions): reject missing VIN pepper before storing vehicle basics`
- `test(functions): cover unauthenticated vehicle-basics submission path`
- `docs(prompts): update prompt 06 to reflect current Join form steps`

## Known limitations

- Detailed vehicle evidence beyond basics, SoH history, and service/fault timeline records
  is not persisted yet.
- Evidence uploads are not implemented (placeholder message shown).
- Admin review queue can read server-side data for admins, but review status updates,
  exports, and moderation actions are not yet implemented.
- Privacy policy is a placeholder — review required before broader evidence data collection.
- Public evidence statistics cover registered cars and SoH history; later evidence metrics
  remain unavailable until their form slices are implemented.
