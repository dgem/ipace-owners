# Operations, CI, and Troubleshooting Prompt

Use this prompt when changing GitHub Actions, Firebase Hosting preview deployments,
production deployment, smoke tests, operational documentation, or passwordless email-link
troubleshooting.

## Goal

Keep local and CI operations reproducible through `make`, avoid skipped or misleading
checks, and preserve the operational lessons learned while moving the project to
Firebase/GCP.

## Command Surface

- The Makefile is the shared command surface for local development and CI.
- `make` and `make help` must list documented targets.
- CI workflows should call Make targets rather than duplicating raw npm, Go, Firebase, or
  gcloud command bodies where a Make target exists.
- Local verification for most changes is `make build` and `make test`.
- Deployment smoke tests require `SMOKE_BASE_URL` and run through `make smoke`.
- `make deploy-functions` should deploy the single Go `Api` Function entrypoint. Avoid
  deploying one Cloud Function per API route because each Gen2 Function deployment triggers
  a separate build and makes PR deployments slow.

## PR Preview Deployment

- Pull requests deploy to Firebase Hosting preview channels in the staging GCP/Firebase
  project.
- Do not use `stage.ipace-owners.org` for PR testing. Use the generated Firebase Hosting
  preview URL, for example `https://ipace-owners-staging--pr-20-abcdef12.web.app`.
- Deploy sequence matters:
  1. Build the site with staging Firebase web config.
  2. Deploy the Firebase Hosting preview channel and extract its generated URL.
  3. Add that preview hostname to Firebase Auth authorized domains, replacing stale PR
     preview domains while preserving permanent domains.
  4. Detect whether backend-related files changed.
  5. Deploy the Go `Api` Function only when backend code, Firebase rewrites, Function env
     generation, Make deploy logic, or deployment workflow files changed.
  6. If `Api` was deployed, refresh the Firebase Hosting preview channel so rewrites point
     at the current Function revision.
  7. Run `make smoke` with `SMOKE_BASE_URL` set to the generated preview URL.
- Serialize staging deployments because PR previews share the staging Auth configuration
  and staging Functions.
- Staging `Api` should accept project-owned Firebase PR preview origins by validated host
  pattern, derive email-link `continueUrl` from that request origin, and omit
  `FIREBASE_EMAIL_LINK_DOMAIN` for previews. This avoids redeploying Functions solely to
  bake a PR-specific preview URL into environment variables.

## Production Deployment

- Merges to `main` deploy production.
- Production Functions should use the verified custom domain for:
  - `ALLOWED_ORIGINS`;
  - `FIREBASE_EMAIL_CONTINUE_URL`;
  - `FIREBASE_EMAIL_LINK_DOMAIN`.
- Production may skip `Api` deployment when backend-related files did not change. Manual
  workflow dispatch should deploy `Api` so operators can force a backend rollout.
- Run `make smoke` directly in the production workflow after Hosting deploy with
  `SMOKE_BASE_URL=https://ipace-owners.org`.

## Smoke Tests

- Smoke tests must run in the workflows that know the real deployed URL: the Firebase
  staging preview workflow and the production deployment workflow.
- Do not rely on GitHub `deployment_status` events for Firebase Hosting smoke tests.
  Firebase preview deployments do not consistently provide a usable target URL through
  those events, which leads to permanently skipped checks.
- The smoke script should reject unrelated hosts such as `github.com` and only allow local,
  project-owned, Firebase Hosting, or Firebase preview hosts.
- The smoke test should verify:
  - the homepage renders project copy;
  - account and vehicle pages use custom passwordless forms, not hosted password widgets;
  - unauthenticated private mutation APIs reject with 401;
  - `GET /api/public-stats` returns successfully.

## Firebase Auth Email Links

- Firebase web API keys are public project identifiers and are expected in Firebase Auth
  action URLs. They are not server credentials.
- The one-time action code (`oobCode`) in an email link is sensitive and must never be
  logged.
- Preview/default `web.app` domains must not be passed as Identity Toolkit `linkDomain`.
  Omit `linkDomain` for PR previews so Firebase uses its default action handler while
  preserving the PR preview URL as `continueUrl`.
- For PR previews, derive `continueUrl` from the validated `Origin` header. Do not accept
  arbitrary `web.app` origins; require the current Firebase project preview-host pattern.
- Production may use the verified custom Hosting domain as `linkDomain` once DNS and
  Firebase Hosting certificate state are active.
- A successful Identity Toolkit response means Firebase accepted the email-link request;
  it does not prove mailbox delivery.
- Delivery troubleshooting should document:
  - checking spam/junk folders;
  - testing another mailbox provider;
  - Firebase Authentication sending quotas and billing plan limits;
  - checking Firebase email template/sender settings;
  - future option to generate action links with Firebase Admin SDK and send through a
    transactional email/SMTP provider if delivery tracking is needed.
- Repeat Join or login requests for the same email must remain account-enumeration safe.
  Logs may include one-way email hashes, masked email addresses, previous Join counts,
  continue hosts, provider status summaries, and response diagnostics, but never raw
  addresses, full provider bodies, or action links.

## Infrastructure Operations

- Use `make infra-config`, `make infra-plan`, and `make deploy-hosting-env` with explicit
  `ENV=staging` or `ENV=production`.
- The infrastructure helpers may initiate `gcloud auth login`, configure local ADC quota
  project, initialise OpenTofu, and select/create the matching workspace.
- DNS remains manually managed at Fasthosts unless the project deliberately migrates
  authoritative DNS to Cloud DNS. OpenTofu should output required Firebase Hosting and ACME
  records but must not attempt unsupported Fasthosts automation.

## Tests

- Keep Node tests that assert smoke tests run in Firebase deploy workflows rather than a
  deployment-status workflow.
- Keep tests for preview URL extraction, preview authorized-domain updates, and Function
  environment generation.
- Run `make build` and `make test` after CI, deployment, or operational prompt changes.
