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
- Local verification for most changes is `make lint`, `make build`, and `make test`.
- Run a separate `Security` workflow on pull requests, pushes to `main`, a weekly Monday
  schedule, and manual dispatch. It must use job-scoped permissions and run CodeQL
  `security-extended` analysis for GitHub Actions, JavaScript/TypeScript, and Go; dependency
  review that rejects newly introduced moderate-or-higher vulnerabilities; `npm audit` with
  a high-severity threshold; and pinned `govulncheck` analysis for reachable Go issues.
- After a Firebase Hosting PR preview passes its smoke test, run a blocking passive OWASP ZAP
  baseline scan against that preview. Use a versioned ZAP container, disable issue creation,
  retain the report as an Actions artifact, and keep reviewed platform/CDN findings in the
  committed `.zap/rules.tsv` baseline so new alert categories still fail the check. Do not run
  active attacks against production or authenticated member data.
- Shared passwordless login forms must declare `method="POST"` and
  `action="/api/send-magic-link"` even though JavaScript normally handles submission, so a
  script failure cannot fall back to a GET request that places a member email in the page URL.
- Dependabot must check npm, Go modules, GitHub Actions, and OpenTofu weekly. Group compatible
  minor and patch updates by ecosystem to reduce PR noise; review major upgrades separately.
- `make lint` is the aggregate source-quality gate. It checks JavaScript, CSS, Markdown,
  JSON/YAML, Nunjucks templates, Bash syntax, Go formatting and vetting, OpenTofu/HCL
  formatting, and SVG/XML syntax; keep focused `lint-*` targets available for iteration.
- Both staging pull-request and production deployment workflows must install OpenTofu and
  run `make lint` after dependency installation and before tests or deployment.
- Keep lint targets self-contained in the declared project toolchains. In particular, SVG/XML
  validation should use the pinned Node dependency rather than assuming runners provide
  `xmllint` or another OS package.
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
- Keep Firebase CLI deployment JSON available for URL extraction and PR diagnostics. If a
  preview deployment fails, CI must print both Firebase CLI stderr and any JSON error payload;
  do not hide the actionable error behind shell redirection.
- Retry Firebase Hosting preview deployment a small, bounded number of times because the
  Firebase CLI performs its own STS exchange and can receive transient connection closures
  even after the workflow's GCP credential check succeeds. Preserve diagnostics from the
  final attempt and still fail deterministically after the retry limit.
- Serialize staging deployments because PR previews share the staging Auth configuration
  and staging Functions.
- Before pushing additional commits to an existing PR branch, cancel any in-progress staging
  deployment for that same branch so obsolete code does not continue through the deployment
  sequence. Keep this cancellation branch-scoped: do not cancel another PR's deployment, and
  do not enable global `cancel-in-progress` while staging infrastructure is shared.
- Authenticate deployments with GitHub OIDC Workload Identity Federation and short-lived
  service-account impersonation. Explicitly generate/export ADC credentials and verify an
  access-token exchange before invoking Firebase CLI; do not introduce long-lived Firebase
  CI tokens or service-account keys.
- Build, test and deploy with the project's current Node Active LTS from `.nvmrc`; do not
  silently downgrade deployment steps to an older Node line. The Firebase CLI transport
  workaround below is scoped to its legacy HTTP client and allows deployment to remain on
  Node 24.
- Reuse the short-lived access token minted by `google-github-actions/auth` through the
  repository Firebase CLI preload helper. This skips Firebase CLI's duplicate STS exchange,
  which currently fails reliably with `Premature close`; the token remains ephemeral and no
  service-account key or legacy Firebase CI token is stored.
- The same preload helper disables response compression in Firebase CLI's legacy `node-fetch`
  transport. GitHub runners have returned prematurely closed compressed responses from STS,
  Cloud Resource Manager and Firebase APIs; requesting identity-encoded responses avoids that
  transport bug without affecting the website's own HTTP behaviour.
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
- Backend change detection must match files beneath `functions/firebase-go/`, not only the
  directory name, so Go changes deploy the Function and refresh preview Hosting rewrites.
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
- Production should use the verified custom Hosting domain as `linkDomain` once DNS and
  Firebase Hosting certificate state are active. DNS cannot issue an HTTP 302 for Firebase
  Auth action links; the action-link host must be a Firebase Hosting/Auth domain accepted
  by Firebase. Keep the sender-domain subdomain separate unless it is also deliberately
  configured as an action-link Hosting domain.
- If `FIREBASE_EMAIL_LINK_DOMAIN` is not present, Functions derive `linkDomain` from an
  HTTPS custom-domain `continueUrl` and suppress it for preview/default Firebase domains,
  localhost, and non-HTTPS URLs.
- Manage the Firebase public-facing project display name with OpenTofu. Firebase's default
  Auth email template inserts that value as `%APP_NAME%`; stale values such as a previous
  product name require an infra apply before new default emails change.
- Resend is the custom email transport for branded passwordless emails. Configure
  `RESEND_API_KEY_<ENV>` as a GitHub environment secret, either manually or by supplying the
  sensitive `resend_api_key` OpenTofu variable during bootstrap with
  `bootstrap_resend_api_key_secret = true`. Leave that boolean false to avoid creating or
  overwriting the GitHub secret. Do not use sensitive values in OpenTofu `for_each` keys;
  use the non-sensitive bootstrap boolean for resource shape and the sensitive variable only
  for the secret value. Manage non-secret `RESEND_FROM_<ENV>`, `RESEND_REPLY_TO_<ENV>`, and
  `RESEND_ASSET_BASE_URL_<ENV>` through OpenTofu/GitHub environment variables. The Function
  sends custom Resend email only when both `RESEND_API_KEY` and `RESEND_FROM` are present;
  otherwise it uses Firebase's default sender. If Resend generation or delivery fails, log a
  sanitized warning and fall back to Firebase default delivery so users still get a sign-in
  link.
- Resend emails should include both HTML and plain text. The HTML email uses the public
  launch hero image at `/images/ipace-hero.png` through an absolute HTTPS asset URL. For PR
  previews, use that PR's generated Firebase Hosting preview origin for assets because the
  image is deployed with the preview branch and the magic link is short-lived. For normal
  staging/production sends, prefer `RESEND_ASSET_BASE_URL_<ENV>` pointing at a stable custom
  domain; avoid localhost and generic Firebase default domains for long-lived email assets.
- OpenTofu can optionally create/read the Resend sending domain with
  `manage_resend_domain = true`, `resend_domain`, `resend_region`, and a Resend API key
  supplied through the sensitive `resend_api_key` variable or `TF_VAR_resend_api_key`.
  Because Fasthosts DNS is not managed by OpenTofu, use the `resend_email_domain` output or
  `make infra-resend-dns-records ENV=<environment>` to copy the required Resend SPF/DKIM/MX
  records into Fasthosts manually. Keep Resend open/click tracking disabled for
  passwordless sign-in emails.
- A successful Identity Toolkit response means Firebase accepted the email-link request;
  it does not prove mailbox delivery.
- Delivery troubleshooting should document:
  - checking spam/junk folders;
  - testing another mailbox provider;
  - Firebase Authentication sending quotas and billing plan limits;
  - checking Firebase email template/sender settings;
  - future option to generate action links with Firebase Admin SDK and send through a
    transactional email/SMTP provider if delivery tracking is needed.
- Firebase Authentication email is part of the user experience, but Firebase's built-in
  passwordless `EMAIL_SIGNIN` body cannot be replaced through the Admin v2 account-management
  templates. Keep the future HTML designs in
  `infra/opentofu/modules/ipace-owners/templates/auth-email` without applying them. A fully
  branded message requires a separately selected transactional provider and server-generated
  action links; do not claim that sender name, reply-to, subject, or body are infrastructure-
  managed while Firebase performs default passwordless delivery.
- Set `firebase_auth_email_domain` to `auth.stage.ipace-owners.org` in staging and
  `auth.ipace-owners.org` in production; keep `firebase_auth_email_action_domain` on the
  environment's Firebase Hosting domain. Add the TXT and CNAME records returned by Firebase
  to Fasthosts, wait for DNS propagation, then rerun
  `make infra-email-domain ENV=<environment>` to complete `VERIFY` and `APPLY`. These sender
  subdomains require no mailbox and no changes to the apex MX records used by Fasthosts
  webmail. Maintain one SPF TXT record per sender subdomain by merging includes instead of
  adding competing SPF records.
- DNS hosting, registration, human mailbox hosting, Firebase Hosting, and authentication-mail
  delivery are independent services. Keep registration, authoritative DNS, and human mail at
  Fasthosts for launch. Moving the whole zone to Cloud DNS or mailboxes to Google Workspace is
  not required for Firebase Auth and must be treated as a separate migration with a complete
  inventory of MX, SPF, DKIM, DMARC, Hosting, and verification records.
- Treat custom sender setup as a two-phase Identity Platform operation. Patch template fields
  without `notification.sendEmail.dnsInfo.useCustomDomain`, initiate `domain:verify` with
  `VERIFY`, and use `APPLY` only after verification succeeds. Setting `useCustomDomain` in the
  initial template update causes `EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED` and fails OpenTofu apply.
- Do not PATCH `notification.sendEmail.callbackUri` when using Firebase's default email
  provider; that field is rejected with the same error. Passwordless Functions set the action
  `linkDomain` and validated `continueUrl` on each email-link request instead.
- Do not PATCH account-action templates while the product uses passwordless email-link sign-in.
  Identity Platform rejects the unrelated reset and verification templates with
  `EMAIL_TEMPLATE_UPDATE_NOT_ALLOWED`. Keep the versioned files as future assets; fully branded
  magic links require server-generated links and a custom transactional delivery service.
- The built-in passwordless `EMAIL_SIGNIN` body cannot be replaced through the Admin v2
  account-management templates. Fully custom sign-in copy requires generated action links
  and a transactional email/SMTP provider; document and secure that provider before making
  such a change.
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
- Run `make lint`, `make build`, and `make test` after CI, deployment, or operational prompt
  changes.
