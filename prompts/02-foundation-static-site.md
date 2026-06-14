# Foundation Static Site Prompt

Build or refine the static foundation for the I-PACE Owners' Advocacy Group website.

## Goal

Create a small, maintainable Eleventy 3 site that can be deployed to Netlify and extended by volunteers. The result should establish the project structure, build scripts, layouts, data files, static assets, and deployment configuration.

## Requirements

- Use Eleventy 3 with `.eleventy.js` in CommonJS format.
- Use `src/` as the Eleventy input directory and `_site/` as output.
- Store page templates directly under `src/`, with nested directories for `src/member/`, `src/admin/`, and `src/updates/`.
- Store shared layouts in `src/_includes/layouts/`.
- Store shared partials in `src/_includes/partials/`.
- Store global data in `src/_data/site.json` and `src/_data/navigation.json`.
- Navigation data should include:
  - Public: Home, About, Evidence, Methodology, FAQ, Updates, Join.
  - Member: Dashboard, My Vehicle, My Evidence, Account.
  - Admin: Review Queue, Submissions, Evidence Files, Public Stats, Exports, Members, Settings.
- Pass through `src/assets/`.
- Pass through `public/` to the site root with `eleventyConfig.addPassthroughCopy({ public: "." })`, so `public/images/example.png` is served at `/images/example.png`.
- Include a favicon in `public/favicon.svg` and link it from the base layout.
- Include a small cookie/privacy notice in the base layout. It should disclose essential services, Netlify Identity storage for sign-in, and the fact that there are no analytics or advertising cookies unless those are later added.
- The cookie/privacy notice should be progressively enhanced: JavaScript may store dismissal
  in browser storage, but users without JavaScript must still see the disclosure and a
  privacy-policy link.
- Add npm scripts:
  - `npm run dev` starts the Eleventy dev server.
  - `npm run build` builds the production site.
  - `npm run clean` removes `_site/`.
- Include `package-lock.json`.
- Configure Netlify in `netlify.toml` with build command `npm run build`, publish directory `_site`, and functions directory `netlify/functions`.
- Add security headers suitable for a static site using Netlify Identity.
- Use GitHub's native automatic Copilot code review via a repository branch ruleset where
  available, rather than adding a custom AI review workflow.

## Deliverables

- Working Eleventy build.
- Base layout with metadata, skip link, header, footer, main landmark, CSS, and deferred scripts.
- Favicon and theme color metadata.
- Cookie/privacy notice markup, no-JavaScript fallback, and dismissal behavior.
- Page layout for standard content pages.
- Form page layout for multi-step forms.
- Update/news collection sorted newest first.
- README describing setup, development, build, deployment, Identity setup, known limitations, and current repo structure.
- README should mention `npx @11ty/eleventy` only as an optional one-off local binary command;
  contributors and Netlify should use npm scripts.
- README should document the native Copilot PR review ruleset if it is enabled for the
  repository.
- `docs/architecture.md` describing future Identity, Functions, Blobs, possible database,
  VIN HMAC, public aggregate data, and server-side JWT verification.

## Validation

- Run `npm run build`.
- Confirm generated pages appear in `_site/`.
- Confirm `npm run dev` starts the Eleventy development server.
- Confirm `npm run clean` removes `_site/`.
- Confirm there is no backend persistence and no real owner data.
