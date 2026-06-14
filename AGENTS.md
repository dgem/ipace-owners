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
├── netlify/functions/   # Netlify Functions directory (placeholder)
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
npm run dev    # Start Eleventy development server (with live reload)
npm run build  # Build production site to _site/
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

## Adding a new update post

1. Create a `.md` file in `src/updates/`.
2. Include front matter: `title`, `date` (YYYY-MM-DD), `summary`, `layout: page.njk`.
3. It will appear automatically on the `/updates/` page.

## Future backend integration

See `docs/architecture.md` for the full intended backend architecture using Netlify
Functions and Netlify Blobs. Key points:

- Forms currently prevent default and show a placeholder message.
- When adding a Function, it goes in `netlify/functions/`.
- Always verify Identity JWTs server-side in Functions.
- Use HMAC with a secret pepper for VIN deduplication (not plain SHA-256).
- Never store raw VINs or personal data in public static files.

## Accessibility

- All pages use semantic HTML (headings, landmarks, labels, fieldsets, legends).
- Focus management is handled in `multistep-form.js` (focus moves to step heading).
- Hidden steps disable focusable elements via `tabindex="-1"`.
- The site respects `prefers-reduced-motion`.
- Colour contrast meets WCAG AA.
- Skip links are included in the base layout.

## Known limitations (current PR)

- Form submissions are not persisted (storage not yet implemented).
- Evidence uploads are not implemented (placeholder message shown).
- Admin review queue is a UI placeholder.
- Privacy policy is a placeholder — review required before live data collection.
- Evidence dashboard shows illustrative data only.
