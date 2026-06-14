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
- **Vanilla JavaScript** — mobile nav, Netlify Identity, multi-step forms
- **[Netlify Identity](https://docs.netlify.com/visitor-access/identity/)** — member authentication
- **[Netlify](https://www.netlify.com/)** — hosting and deployment

---

## Local setup

### Prerequisites

- Node.js 18+ (20 recommended)
- npm 9+

### Install

```bash
npm install
```

### Development server

```bash
npm run dev
```

Starts the Eleventy dev server at `http://localhost:8080` with live reload.

You can also run the local Eleventy binary directly with:

```bash
npx @11ty/eleventy --serve
```

But contributors and Netlify should use the `npm run dev` / `npm run build` scripts.

### Production build

```bash
npm run build
```

Output is written to `_site/`.

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
      identity.js        # Netlify Identity integration
      multistep-form.js  # Multi-step form controller
public/images/     # Static images
netlify/functions/ # Netlify Functions (placeholder — not yet implemented)
docs/
  architecture.md  # Future architecture documentation
prompts/           # Sequenced prompts for rebuilding and evolving the product
.eleventy.js       # Eleventy configuration
netlify.toml       # Netlify configuration
```

---

## Netlify deployment

1. Fork or clone this repository.
2. Create a new site in Netlify connected to your repository.
3. Build settings are pre-configured in `netlify.toml`:
   - **Build command:** `npm run build`
   - **Publish directory:** `_site`
   - **Functions directory:** `netlify/functions`

### Enable Netlify Identity

After deploying to Netlify, you **must** enable Netlify Identity manually:

1. Go to your site in the Netlify dashboard.
2. Navigate to **Site configuration → Identity**.
3. Click **Enable Identity**.
4. Configure registration settings:
   - For a closed group, set **Registration preferences** to **Invite only**.
   - For open registration, leave as **Open**.
5. Optional: configure external OAuth providers (Google, GitHub) if desired.

> The Netlify Identity widget is loaded from `https://identity.netlify.com/v1/netlify-identity-widget.js`
> and will not function until Identity is enabled in the Netlify UI.

### Admin role assignment

To grant a member admin access:

1. Go to **Site configuration → Identity → Users**.
2. Select the user.
3. Edit their metadata and add `roles: ["admin"]` to `app_metadata`.

---

## Known limitations

The following features are **not yet implemented** in this version:

- **Form submission persistence** — Forms show a placeholder "not yet active" message.
  No data is sent or stored. Backend implementation via Netlify Functions is planned.
- **Evidence document uploads** — A placeholder message explains what will be supported.
  Requires Netlify Blobs + Functions integration.
- **Admin review queue** — UI placeholder only. No data is accessible from the admin pages.
- **Privacy policy** — The current policy is a placeholder. A formal policy is required
  before live personal data is collected.
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
- `identity.js` — Netlify Identity init and UI state
- `multistep-form.js` — generic multi-step form (data-attribute driven)

### Adding pages

Add `.md` or `.njk` files to `src/`. Use `layout: page.njk` for standard pages
or `layout: form-page.njk` for form pages (adds `multistep-form.js`).

### Adding updates

Add `.md` files to `src/updates/` with `title`, `date` and `summary` front matter.
Posts appear automatically on `/updates/`.

### Future backend

See `docs/architecture.md` for the intended architecture using Netlify Functions and
Netlify Blobs for data persistence and admin features.

### Prompt maintenance

Product-generation prompts live in `prompts/`. They are sequenced with two-digit prefixes
so the project can be rebuilt or extended in a controlled order:

- `00-original-project-prompt.md` preserves the initial generation prompt.
- Later files split the product into foundation, design, content, identity, forms,
  evidence dashboard, and backend-roadmap concerns.

When adding or refining prompts, keep the numeric prefix, make the prompt independently
usable, and avoid duplicating live implementation details that belong in README or
`docs/architecture.md`.

---

## Contributing

This site is maintained by volunteers. If you have skills in web development, data, legal,
or consumer rights and want to help, please [contact us](/contact/) or indicate your interest
when joining the group.

---

## Licence

Content and code are copyright the I-PACE Owners' Advocacy Group contributors.
No JLR / Jaguar trademarks, logos, or copyrighted imagery are used.
