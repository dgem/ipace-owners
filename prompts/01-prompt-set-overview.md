# Prompt Set Overview

Use these prompts as a maintainable product blueprint. They are ordered so a fresh agent can rebuild the project from nothing, then progressively add product features without losing the existing constraints.

## How to use this directory

1. Start with `00-original-project-prompt.md` for historical context only.
2. Use `02-foundation-static-site.md` to scaffold or re-scaffold the Eleventy site.
3. Use `03-design-system-and-accessibility.md` whenever layout, CSS, components, or accessibility behavior changes.
4. Use `04-public-content-and-positioning.md` for public pages and copy.
5. Use `05-identity-member-admin-gating.md` for Firebase Auth and protected member/admin pages.
6. Use `06-forms-and-evidence-collection.md` for join and vehicle data form UX.
7. Use `07-evidence-dashboard-and-methodology.md` for public aggregate evidence views and methodology copy.
8. Use `08-backend-security-and-storage.md` before changing persistence, stored record shapes, or backend security boundaries.
9. Use `09-architecture-overview.md` for cross-cutting architecture, security boundaries, and data-flow constraints.
10. Use `10-functions-shared-utilities.md` for shared Go Cloud Function helpers.
11. Use `11-functions-identity-and-join.md` for magic-link and Join submission flows.
12. Use `12-functions-vehicle-basics.md` for the signed-in vehicle-basics slice.
13. Use `13-functions-member-admin-data.md` for member/admin data access Functions.
14. Use `14-functions-future-evidence-and-stats.md` for future evidence, uploads, review
    mutations, and extensions to the implemented public statistics path.
15. Use `15-firestore-static-json-data-model.md` for the canonical Firestore model and JSON snapshot strategy.
16. Use `16-member-vehicle-workspace-and-service-history.md` for the tabbed member vehicle
    dashboard, SoH graph, and editable service/fault timeline.
17. Use `17-operations-ci-and-troubleshooting.md` for CI/CD, preview deployments, smoke
    tests, Firebase Auth delivery troubleshooting, and operational runbooks.
18. Use `18-stakeholder-feedback.md` as the current stakeholder-feedback intake log.
19. Use `19-launch-readiness.md` for public launch preparation, launch-week checks,
    passwordless email delivery, DNS/auth readiness, data protection, and rollback planning.
20. Use `20-clean-room-reconstruction-contract.md` as the final authority for a from-scratch
    rebuild, exact route/API/data inventories, acceptance criteria, asset preservation, and
    the boundary between source reconstruction and production disaster recovery.

## Global constraints for every prompt

- Use Eleventy 3, Nunjucks, Markdown, custom CSS, and vanilla JavaScript.
- Do not add Tailwind, Bootstrap, React, Vue, Svelte, TypeScript, or a bundler.
- Keep the public site static unless a prompt explicitly asks for Go Cloud Functions.
- Do not commit or generate real owner data, raw VINs, private evidence files, or personal information.
- Do not use Jaguar/JLR logos, trademarks as branding, or copyrighted imagery.
- Maintain a constructive, serious, evidence-led tone.
- Keep all UI mobile-first, semantic, keyboard accessible, and WCAG AA conscious.
- Run `make build` after implementation and fix any build failures.
- Add or update tests for behavioural changes and run `make test`.
- Pressure-test changes locally before opening a pull request.
- Submit all changes through PRs with clear descriptions, verification steps, and test notes.
- Require code review before merge (Copilot review where enabled, plus human review).
- Use semantic commit messages in `type(scope): description` format.
- Treat `00-original-project-prompt.md` as immutable source history. Update the numbered
  prompts from `01-` onward when requirements evolve.
- Treat `00-original-project-prompt.md` as non-authoritative whenever it conflicts with
  prompts `01-20`; later specific prompts override earlier general prompts.
- Keep prompts aligned with implementation and README so the project can be recreated from
  prompt files plus README alone.
- Treat durable knowledge from user conversations as source material for these prompts.
  When a debugging session, deployment attempt, UX review, or architecture discussion
  reveals a constraint future agents need, encode it in the relevant numbered prompt during
  the same PR. Do not leave important decisions only in chat context.
- Before closing a task, check whether any prompt needs updates for changed behaviour,
  operational lessons, CI/deployment rules, data model decisions, security/privacy
  constraints, or UX/content direction. If the knowledge does not fit an existing prompt,
  create or split a sequenced prompt by concern.
