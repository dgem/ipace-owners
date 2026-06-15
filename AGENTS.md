# AGENTS.md — I-PACE Owners' Advocacy Group

## Project overview

This is the I-PACE Owners' Advocacy Group website — a static, mobile-first, accessible
public site built with Eleventy (11ty) and deployed to Netlify.

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
│           ├── identity.js        # Netlify Identity UI integration
│           └── multistep-form.js  # Generic multi-step form controller
├── public/images/       # Static images (passed through to _site)
├── netlify/functions/   # Netlify Functions
│   ├── lib/             # Shared Function utilities
│   ├── send-magic-link.js
│   ├── submit-join.js
│   └── submit-vehicle-basics.js
├── docs/
│   └── architecture.md  # Future architecture documentation
├── prompts/             # Sequenced prompts for rebuilding and evolving the product
├── .eleventy.js         # Eleventy configuration
├── netlify.toml         # Netlify build, redirect and header configuration
├── package.json         # npm scripts and dependencies
├── AGENTS.md            # This file
└── README.md            # Setup and deployment documentation
```

## Development commands

```bash
npm install    # Install dependencies
npm run dev    # Start Netlify Dev with Eleventy and Functions
npm run dev:eleventy # Start Eleventy only, without Netlify Functions
npm run build  # Build production site to _site/
npm test       # Run Node tests for Functions and critical form wiring
npm run clean  # Remove _site/ directory
```

Eleventy v3 is used. The Eleventy config file is `.eleventy.js` (CommonJS format).

## Technology choices

- **Static site generator:** Eleventy 3 (`@11ty/eleventy`)
- **Templates:** Nunjucks (`.njk`) for complex pages; Markdown for content pages
- **CSS:** Custom CSS only — no Tailwind, no Bootstrap, no utility frameworks
- **JavaScript:** Plain vanilla JS — no React, Vue, Svelte, or heavy frameworks
- **Authentication:** Netlify Identity (`netlify-identity-widget` CDN)
- **Hosting:** Netlify
- **Backend (future):** Netlify Functions + Netlify Blobs

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

## Authentication (Netlify Identity)

- Identity widget is loaded from the CDN in `base.njk`.
- `identity.js` initialises the widget and updates the header UI.
- On Join form completion, `identity.js` calls `POST /.netlify/functions/submit-join`
  once. The function stores the Join submission and dispatches a Netlify Identity
  confirmation or recovery email (magic sign-in link) server-side.
- Existing users can request another magic link through `POST /.netlify/functions/send-magic-link`
  without resubmitting the Join form.
- Signed-in vehicle basics are stored by `POST /.netlify/functions/submit-vehicle-basics`.
- Member pages use `data-auth-gate` / `data-auth-content` attributes.
- Admin pages use `data-admin-gate` / `data-admin-content` attributes.
- **Frontend gating is not sufficient for real data access.** Future Netlify Functions
  must verify Identity JWTs server-side.

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
- Include `09-architecture-overview.md` in the maintained prompt set and keep it aligned
  with `docs/architecture.md`.
- **Keep prompts in sync with the actual state of the project.** After implementing a
  feature, update the relevant prompt file to reflect what was built so that the project
  can be recreated from the prompts and README alone. If behaviour changes (e.g., a new
  Function is added, a form field is removed), update the corresponding prompt.

## Testing

- **Tests are required for all behavioural changes.** Any change to Netlify Functions,
  form submission wiring, Identity handoff, or shared utilities must include or update
  Node tests in `test/`.
- Run `npm test` before considering a change complete. All existing tests must pass.
- Tests should cover:
  - Server-side validation and authorization paths (unauthenticated, authenticated, admin).
  - Input sanitisation and edge cases (empty bodies, invalid JSON, honeypot fields).
  - Storage-shaping logic (record structure, metadata, HMAC behaviour).
  - Magic-link handoff behaviour (new vs existing user flow).
- Content-only changes (Markdown copy, CSS, static templates) need not add tests but must
  pass `npm run build`.

## Code review and pull requests

- **All changes must be submitted via pull requests.** Do not push directly to the default
  branch.
- Every PR must include a clear description covering:
  - What changed and why.
  - Which files were added or modified.
  - How to verify the change locally (e.g., `npm run dev` and navigate to X).
  - Whether tests were added or updated.
- **Code review is required before merging.** Use GitHub's automatic Copilot code review
  (configured via repository branch ruleset) as a first pass, but every PR must receive
  human review for logic, security, accessibility, and tone.
- Do not merge until `npm run build` and `npm test` both pass cleanly.

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

- Full vehicle/evidence form submissions are not persisted yet.
- Evidence uploads are not implemented (placeholder message shown).
- Admin review queue is a UI placeholder.
- Privacy policy is a placeholder — review required before broader evidence data collection.
- Evidence dashboard shows illustrative data only.
