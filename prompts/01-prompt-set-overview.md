# Prompt Set Overview

Use these prompts as a maintainable product blueprint. They are ordered so a fresh agent can rebuild the project from nothing, then progressively add product features without losing the existing constraints.

## How to use this directory

1. Start with `00-original-project-prompt.md` for historical context only.
2. Use `02-foundation-static-site.md` to scaffold or re-scaffold the Eleventy site.
3. Use `03-design-system-and-accessibility.md` whenever layout, CSS, components, or accessibility behavior changes.
4. Use `04-public-content-and-positioning.md` for public pages and copy.
5. Use `05-identity-member-admin-gating.md` for Netlify Identity, member pages, and admin placeholders.
6. Use `06-forms-and-evidence-collection.md` for join and vehicle data form UX.
7. Use `07-evidence-dashboard-and-methodology.md` for public aggregate evidence views and methodology copy.
8. Use `08-backend-roadmap-functions-blobs.md` before implementing real persistence.
9. Use `09-architecture-overview.md` for cross-cutting architecture, security boundaries, and data-flow constraints.

## Global constraints for every prompt

- Use Eleventy 3, Nunjucks, Markdown, custom CSS, and vanilla JavaScript.
- Do not add Tailwind, Bootstrap, React, Vue, Svelte, TypeScript, or a bundler.
- Keep the site static unless a prompt explicitly asks for Netlify Functions.
- Do not commit or generate real owner data, raw VINs, private evidence files, or personal information.
- Do not use Jaguar/JLR logos, trademarks as branding, or copyrighted imagery.
- Maintain a constructive, serious, evidence-led tone.
- Keep all UI mobile-first, semantic, keyboard accessible, and WCAG AA conscious.
- Run `npm run build` after implementation and fix any build failures.
- Add or update tests for behavioural changes and run `npm test`.
- Pressure-test changes locally before opening a pull request.
- Submit all changes through PRs with clear descriptions, verification steps, and test notes.
- Require code review before merge (Copilot review where enabled, plus human review).
- Use semantic commit messages in `type(scope): description` format.
- Treat `00-original-project-prompt.md` as immutable source history. Update the numbered
  prompts from `01-` onward when requirements evolve.
- Keep prompts aligned with implementation and README so the project can be recreated from
  prompt files plus README alone.
