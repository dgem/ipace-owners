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
- Pass through `src/assets/`.
- Pass through `public/` to the site root with `eleventyConfig.addPassthroughCopy({ public: "." })`, so `public/images/example.png` is served at `/images/example.png`.
- Include a favicon in `public/favicon.svg` and link it from the base layout.
- Add npm scripts:
  - `npm run dev` starts the Eleventy dev server.
  - `npm run build` builds the production site.
  - `npm run clean` removes `_site/`.
- Include `package-lock.json`.
- Configure Netlify in `netlify.toml` with build command `npm run build`, publish directory `_site`, and functions directory `netlify/functions`.
- Add security headers suitable for a static site using Netlify Identity.

## Deliverables

- Working Eleventy build.
- Base layout with metadata, skip link, header, footer, main landmark, CSS, and deferred scripts.
- Favicon and theme color metadata.
- Page layout for standard content pages.
- Form page layout for multi-step forms.
- Update/news collection sorted newest first.
- README describing setup, development, build, deployment, Identity setup, known limitations, and current repo structure.

## Validation

- Run `npm run build`.
- Confirm generated pages appear in `_site/`.
- Confirm there is no backend persistence and no real owner data.
