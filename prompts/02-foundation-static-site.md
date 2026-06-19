# Foundation Static Site Prompt

Build or refine the static foundation for the I-PACE Owners' Advocacy Group website.

## Goal

Create a small, maintainable Eleventy 3 site that can be deployed to Firebase Hosting and extended by volunteers. The result should establish the project structure, build scripts, layouts, data files, static assets, and deployment configuration.

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
- Include a small cookie/privacy notice in the base layout. It should disclose essential services, Firebase Authentication storage for sign-in, and the fact that there are no analytics or advertising cookies unless those are later added.
- The cookie/privacy notice should be progressively enhanced: JavaScript may store dismissal
  in browser storage, but users without JavaScript must still see the disclosure and a
  privacy-policy link.
- Add npm scripts:
  - `npm run dev` starts the local development server.
  - `npm run dev:eleventy` starts the plain Eleventy dev server for static-only debugging.
  - `npm run build` builds the production site.
  - `npm run clean` removes `_site/`.
- Add a `Makefile` as the shared local and CI command surface:
  - `make` and `make help` list available targets extracted from target descriptions.
  - `make install`, `make dev`, `make dev-eleventy`, `make build`, `make clean`,
    `make test-node`, `make test-go`, and `make test` delegate to the underlying npm/Go
    commands.
  - `make functions` lists deployed Cloud Function entrypoints.
  - Deployment workflows should call Make targets rather than duplicating raw npm, Go,
    Firebase, or gcloud command bodies.
- Include `package-lock.json`.
- Include `firebase-tools` as a development dependency so deployments and preview channels are repeatable.
- Configure Firebase Hosting in `firebase.json` with publish directory `_site`, security
  headers, and `/api/*` rewrites to Go Cloud Functions.
- Add security headers suitable for a static site using Firebase Authentication.
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
- README describing setup, development, build, Firebase/GCP deployment, Identity setup, known limitations, and current repo structure.
- README should document Makefile targets as the preferred local development and CI
  command surface.
- README should explain Firebase build-time config and local/static debugging.
- README should document the native Copilot PR review ruleset if it is enabled for the
  repository.
- `prompts/09-architecture-overview.md` describing Firebase Auth, Go Functions, Firestore,
  generated JSON snapshots, Cloud Storage for uploaded files, VIN HMAC, public aggregate
  data, and server-side ID-token verification.

## Validation

- Run `make build`.
- Confirm generated pages appear in `_site/`.
- Confirm `make` lists available targets.
- Confirm `make dev` starts the documented local development server.
- Confirm `make dev-eleventy` starts the Eleventy development server.
- Confirm `make clean` removes `_site/`.
- Confirm implemented persistence is limited to Join submissions and signed-in vehicle
  basics, with no real seed/test owner data committed to the repository.
