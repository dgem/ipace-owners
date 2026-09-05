# Operations, CI, and Troubleshooting Prompt

Use this prompt when changing GitHub Actions, Firebase Hosting preview deployments,
production deployment, smoke tests, operational documentation, or passwordless email-link
troubleshooting.

## Goal

Keep local and CI operations reproducible through `make`, avoid skipped or misleading
checks, and preserve the operational lessons learned while moving the project to
Firebase/GCP.

The `/admin/email-campaigns/` Join re-engagement control uses the CLI's consent and registration
suppression boundary. Preview exposes aggregate counts and the exact HTML delivery in a sandboxed
iframe, with the plain-text alternative available on demand and a safe link placeholder, but no
recipient data. Send controls stay visible but disabled until the preview succeeds. Sending
requires the current campaign ID, exact audience count, and `SEND <count>` confirmation; each request sends at most 100 messages
and records a hashed Firestore delivery ledger with Resend idempotency keys. Re-preview between
batches and stop to investigate any provider or ledger error.

The separate member-referral campaign targets only Firebase accounts with a matching
contact-consenting Join record. Its live copy reports progress to 1,000 and the exact doubled
total if every current owner finds one more. Include monochrome actions for Facebook, X,
Bluesky, LinkedIn, Instagram, WhatsApp, and email; Instagram links to the group's
`@ipaceowners` profile because it has no reliable web share composer. Display a ready-to-copy
“I-PACE owners are stronger together” suggested post with a direct Join CTA, and prefill it in
share URLs where the platform supports composer text. Apply the same preview,
confirmation, batch, registration
recheck, and hashed-ledger controls as registration reminders. Keep the embedded Markdown prose
editable: regression tests protect complete template-field substitution, shared email chrome,
destinations and delivery safeguards without pinning editorial sentences or chosen numeric fields.

The `Reach 1,000` campaign targets all contact-consenting Join records, whether their Firebase
email sign-in was completed or not, deduped by canonical email. Its embedded Markdown thanks
members for joining and supporting the group, says it launched on 17 July less than two weeks
earlier, reports the live joined total, explains the formal approach to Jaguar about members'
shared concerns and asks Jaguar to engage constructively on options for everyone, links the cited
I-PACE population source, and asks recipients to recruit by sharing. Its subject must lead with
thanks for joining before the 1,000-member CTA. Do not require Auth
registration for this audience. Apply the same preview, exact count confirmation, 100-message
batch, hashed delivery ledger and provider-idempotency controls.

The same page provides custom verified-member campaigns. Preview validates an allowlisted
`{{name}}` substitution language rather than executing arbitrary Go templates, saves a Firestore
draft, calculates the canonical-email-deduped intersection of verified Auth accounts and
contact-consenting Join records, and renders branded HTML in a sandbox plus plain text. Sending
loads the saved immutable content, rechecks the audience and `SEND <count>` confirmation, and
uses the same 100-message hashed/idempotent batches. Parent campaign documents retain aggregate
eligible, sent, failed-attempt, remaining, batch and timestamp history. Write them as complete
struct replacements; never pass a Go struct with Firestore `MergeAll`, which accepts map data
only. Replacing the parent document must leave its hashed delivery subcollection intact so a
post-delivery summary retry cannot resend recipients. “Tweak and rerun” clones
content into a new run; never edit a run after delivery starts. History may infer old specialised
runs from legacy delivery-only subcollections, where only the sent count is recoverable.

The fixed September Survey invitation is a group-wide member poll rather than an ad-hoc contact
campaign. Its audience is every enabled Firebase Auth account with an email address, regardless
of whether a legacy Join record can be matched; use the Auth creation time and display name where
that Join data is unavailable. This deliberate exception must remain scoped to the survey tool.
Refreshing history must reconcile stored Resend IDs with the paginated sent-email API, cache
provider checks for five minutes, and aggregate delivered, awaiting-delivery, opened, clicked,
delayed, bounced, suppressed, complained, provider-failed and combined-undeliverable outcomes
into history and the Admin home. Ignore provider recipient fields: persist only normalised status
and update time on hashed delivery documents and never return addresses to the browser.
Registration reminders remain rerunnable only through their specialised tool because they require
an unverified-member audience and a fresh private sign-in link; do not clone them into the
verified-member custom editor. Other historical runs with saved subject and Markdown, including
JLR Contact, use “Tweak and rerun” to create a linked Freeform run. An unsent editable draft uses
“Edit draft” instead. Keep only headline history metrics visible and retain the full set, including
zero values, in an expandable “more metrics” control.
Allow drafts to reopen and partially sent custom runs to continue after re-previewing the exact
unchanged saved content. If the administrator edits a run that has sent anything, clear its active
campaign ID and create a new run linked through `sourceCampaignId`.

Campaign body copy is source-controlled Markdown (`.md`, including front matter). The specialised registration reminder,
member-referral, and Reach 1,000 campaigns render their deployed Markdown templates from
`functions/firebase-go/email-templates/`; their delivery mechanics remain in Go. Custom-editor
campaigns are also Markdown files with front matter in that directory. Every static campaign
must have a dedicated tab with the same preview, exact-count confirmation, and bounded-send flow;
the server selects its Markdown source and audience, never JavaScript. The JLR Contact campaign
for the 16 August 2026 update is one of those dedicated tabs. Once a static JLR batch has sent,
preserve its saved subject, body and hero for subsequent previews even if its Markdown source is
later edited; show the saved email and a disabled send action when the audience is exhausted,
rather than failing the preview. Freeform remains only for ad-hoc,
editable verified-member campaigns. Its copy must describe the proposed early-September
face-to-face meeting with JLR's UK Director of Client Care at Gaydon, while making the date and
venue subject to final confirmation and distinguishing it from a substantive JLR commitment. Link
to `/updates/jaguar-contact/`, ask owners to help reach 1,000 members, and invite registered
members to add vehicle, SoH, service, fault and repair records before September so the group can
reference suitably anonymised evidence. Its `heroImage` and `heroImageAlt` Markdown front-matter
fields use the same bright, non-branded `/images/jlr-client-care-september-hero.png` holiday image
as the public update, rather than the generic campaign hero. Static template heroes must be
site-owned `/images/` paths; Freeform campaigns retain the generic hero.

## Command Surface

- The Makefile is the shared command surface for local development and CI.
- `make` and `make help` must list documented targets.
- CI workflows should call Make targets rather than duplicating raw npm, Go, Firebase, or
  gcloud command bodies where a Make target exists.
- Local verification for most changes is `make lint`, `make build`, and `make test`.
- Layout/navigation changes also require `make dev` plus `make test-visual`. The staging validate
  job runs deterministic Chrome checks at desktop and mobile viewports and uploads screenshots
  even on failure. Keep authenticated visual fixtures credential-free and assert geometry and
  visibility rather than relying only on pixel snapshots.
- Run a separate `Security` workflow on pull requests, pushes to `main`, a weekly Monday
  schedule, and manual dispatch. It must use job-scoped permissions and run CodeQL
  `security-extended` analysis for GitHub Actions, JavaScript/TypeScript, and Go; dependency
  review that rejects newly introduced moderate-or-higher vulnerabilities; `npm audit` with
  a high-severity threshold; and pinned `govulncheck` analysis for reachable Go issues.
- If a newly disclosed npm advisory affects transitive packages while the maintained direct
  tool is already current, prefer narrow pinned `overrides` for patched transitive releases
  over downgrading the direct tool. Regenerate `package-lock.json`, require a zero-result
  full npm audit, and exercise the affected build or deployment CLI before pushing.
- After a Firebase Hosting PR preview passes its smoke test, run a blocking passive OWASP ZAP
  baseline scan against that preview. Use a versioned ZAP container, disable issue creation,
  retain the report as an Actions artifact, and keep reviewed platform/CDN findings in the
  committed `.zap/rules.tsv` baseline so new alert categories still fail the check. Do not run
  active attacks against production or authenticated member data.
- Shared passwordless login forms must declare `method="POST"` and
  `action="/api/send-magic-link"` even though JavaScript normally handles submission, so a
  script failure cannot fall back to a GET request that places a member email in the page URL.
- Preview administrators may be provisioned directly in Firebase Authentication without a
  copied Join submission. `SendMagicLink` therefore accepts either registration source while
  keeping its public response generic. When troubleshooting a generic success with no email,
  inspect the structured `registrationSource`, Join/Auth lookup warnings and provider handoff
  events; never expose those distinctions in the browser response.
- Dependabot must check npm, Go modules, GitHub Actions, and OpenTofu weekly. Group compatible
  minor and patch updates by ecosystem to reduce PR noise; review major upgrades separately.
- Before declaring a PR ready, inspect every current human and automated review thread against
  the latest commit. Implement valid findings, and record any stale or intentionally declined
  finding with its evidence; do not assume a Copilot, CI, or human comment is correct without
  verification.
- `make lint` is the aggregate source-quality gate. It checks JavaScript, CSS, Markdown,
  JSON/YAML, Nunjucks templates, Bash syntax, Go formatting and vetting, OpenTofu/HCL
  formatting, and SVG/XML syntax; keep focused `lint-*` targets available for iteration.
- Both staging pull-request and production deployment workflows must install OpenTofu and
  run `make lint` after dependency installation and before tests or deployment.
- Pin third-party Actions to immutable commit SHAs while retaining the release tag in a
  comment for Dependabot and human readability.
- Keep pull-request validation separate from privileged staging deployment. Every PR may run
  the read-only lint, test, and build job after any GitHub-required external-contributor
  approval, but automatically deploy a preview only when the PR author is the repository
  owner and the head repository is this repository. Other PRs stop after validation. A fork
  must never receive OIDC, write-capable pull-request permissions, or staging secrets.
- Configure GitHub Actions to require workflow approval for every external contributor, not
  only first-time contributors. Keep automatic staging authorization in the workflow's
  repository-owner and same-repository job condition; do not rely only on environment state.
- Keep lint targets self-contained in the declared project toolchains. In particular, SVG/XML
  validation should use the pinned Node dependency rather than assuming runners provide
  `xmllint` or another OS package.
- Deployment smoke tests require `SMOKE_BASE_URL` and run through `make smoke`.
- `make deploy-functions` should deploy the single Go `Api` Function entrypoint. Avoid
  deploying one Cloud Function per API route because each Gen2 Function deployment triggers
  a separate build and makes PR deployments slow. Keep the Function timeout at 180 seconds so
  the explicitly confirmed Instagram publish route can wait for bounded Meta container
  processing without relying on the platform default timeout.

## PR Preview Deployment

- Repository-owner same-repository pull requests deploy automatically to Firebase Hosting
  preview channels in the staging GCP/Firebase project. All other PRs stop after validation.
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
  7. Run `make smoke` with `SMOKE_BASE_URL` set to the generated preview URL. The public-stats
     request must require the current schema and headline aggregate, causing an outdated
     stored snapshot to regenerate under the Function runtime identity.
- Keep Firebase CLI deployment JSON available for URL extraction and PR diagnostics. If a
  preview deployment fails, CI must print both Firebase CLI stderr and any JSON error payload;
  do not hide the actionable error behind shell redirection.
- Retry Firebase Hosting preview deployment a small, bounded number of times because the
  Firebase CLI performs its own STS exchange and can receive transient connection closures
  even after the workflow's GCP credential check succeeds. Preserve diagnostics from the
  final attempt and still fail deterministically after the retry limit.
- Serialize staging deployments because PR previews share the staging Auth configuration
  and staging Functions.
- When a repository-owner pull request is merged, trigger a closed-PR cleanup job that uses the
  same staging deployment credentials and serialized concurrency group to delete exactly its
  `pr-<number>` Firebase Hosting preview channel. Do not delete previews for closed-but-unmerged
  pull requests or external contributors.
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
  Node 24. Make every local npm/Node-backed Make target depend on a shared runtime check that
  compares the active Node major with `.nvmrc` and tells the operator to run `nvm use` when
  they differ. GitHub Actions must continue to configure Node from the same `.nvmrc` file.
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
- Serialize production deployment jobs without cancellation. Closely timed merges must wait
  for the active deploy rather than racing Cloud Functions operations and failing with
  `409 unable to queue the operation`.
- Production Functions should use the verified custom domain for:
  - `ALLOWED_ORIGINS`;
  - `FIREBASE_EMAIL_CONTINUE_URL`;
  - `FIREBASE_EMAIL_LINK_DOMAIN`.
- Production may skip `Api` deployment when backend-related files did not change. Manual
  workflow dispatch should deploy `Api` so operators can force a backend rollout.
- Backend change detection must match files beneath `functions/firebase-go/`, not only the
  directory name, so Go changes deploy the Function and refresh preview Hosting rewrites.
- Require the production smoke test to receive the current public-statistics schema and
  headline aggregate. `PublicStats` regenerates an outdated stored snapshot under the
  Function runtime identity; do not grant the GitHub deployer direct member-data access.
- Run `make smoke` directly in the production workflow after Hosting deploy with
  `SMOKE_BASE_URL=https://ipace-owners.org`.

## Production Monitoring and Recovery

- Manage the production operations dashboard, external uptime checks, notification channel,
  and alert policies in OpenTofu; do not create one-off console-only checks or alerts. Monitoring
  is opt-in per environment so staging remains quiet by default.
- The production baseline independently checks `https://ipace-owners.org/` and
  `https://ipace-owners.org/api/public-stats` from Europe every five minutes. The latter catches
  Hosting rewrite and public `Api` failures which a static homepage check cannot see. A policy
  opens after ten minutes of continued failure and auto-closes after thirty minutes without a
  signal; this deliberately avoids waking operators for a single transient probe failure.
- Set `monitoring_alert_email` to a real operational mailbox before applying production so the
  managed notification channel can deliver incidents. Leaving it empty keeps the dashboard and
  incidents but sends no email. Do not configure an individual member address without their
  agreement.
- Investigate an availability alert in this order: the dashboard’s check result, the serialized
  production deployment run and smoke result, then Cloud Logging for the `Api` Cloud Run/Function
  revision. Firebase Hosting serves the prior release until a new release is complete; an
  unsuccessful smoke test does not automatically roll back a completed release. Roll back the
  Hosting release in Firebase Hosting release history after confirming the previous release is
  healthy, then use the dashboard and smoke test to confirm recovery.
- The dashboard’s request-rate chart is scoped to the Gen 2 `Api` service in the configured
  region. It is operational context, not a privacy analytics product: do not enable Firebase
  Hosting request logging merely to populate it, as that has separate privacy and cost review.

## Human-controlled Facebook outreach

- Provide an admin-only `/admin/outreach/` workspace for generating user-initiated Facebook
  search links from editable phrases and group URLs, plus issue-specific editable reply drafts.
- The browser helper must never fetch or scrape Facebook, operate a logged-in Facebook session,
  open searches automatically, send direct messages, submit posts or retain Facebook content.
- Accept only HTTPS `facebook.com/groups/<id-or-slug>` URLs, discard extra paths and parameters,
  deduplicate inputs, URL-encode every query, and mark explicit outbound links `noopener` and
  `noreferrer`.
- Keep every response human-reviewed and manually posted. Replies should offer useful evidence-
  gathering steps, disclose the volunteer connection when inviting participation, avoid diagnosis
  or official-advice claims, follow group rules, and stop on objection.

## Veo generation operations

- Keep the Function and private campaign-media bucket in `europe-west2`, but configure Veo 3.1
  with the explicit `us-central1` endpoint. Reject the `global` routing alias so administrators
  are not given a false impression of UK-resident generation.
- Provision `aiplatform.googleapis.com` and its managed service identity through OpenTofu before
  granting the intended service-agent role and bucket-scoped object access. Do not wait for the
  first billable generation request to trigger service-agent creation.
- A `Veo generation started` application log proves that Vertex accepted an asynchronous
  operation, not that media rendering completed. Poll the stored operation to its terminal state.
- If Vertex returns code `9` while service agents are being provisioned, apply the current
  infrastructure, allow IAM propagation to complete, and start a new explicitly confirmed job.
  Failed billable jobs are immutable and must not be silently retried.
- Store and log only an allowlisted failure classification with provider numeric code/status.
  Show administrators an actionable safe message, and do not expose arbitrary provider text to
  the browser.

## Firebase administrator reconciliation

- Treat OpenTofu's administrator email map as authoritative whenever management is enabled. The
  shared module must always add `dan@kanzi.co.uk` to the desired set.
- The Google provider has no Firebase Auth user data source. Use the tested reconciliation bridge
  to page through Identity Platform accounts, resolve email addresses to environment-specific UIDs,
  merge `admin: true` without discarding unrelated claims, and remove only admin access from users
  no longer configured.
- Refuse an empty desired set, duplicate/invalid emails and missing Firebase accounts. Never print
  user email addresses, UIDs or claim contents during a successful reconciliation.
- Reconcile staging and production independently. Users need a newly issued ID token after a claim
  change. Disabling management leaves existing claims untouched, so revoke removed admins first.
- Show desktop and mobile admin navigation only after the signed-in user's ID-token claims contain
  `admin: true` or an `admin` role. Keep server-side `AdminData` verification authoritative.

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
  Staging uses `https://ipace-owners.org` because `stage.ipace-owners.org` is the Resend sender
  domain and does not serve Firebase Hosting assets.
- Keep Join re-engagement as an operator CLI, not an HTTP Function. The CLI must select staging or
  production explicitly, extract Join and Firebase Auth data directly, and run a no-send comparison
  by default. Suppress exact email, plus-address canonical email, and normalised display-name
  matches. Generate a stable environment/date campaign identifier when omitted; require `--send`,
  the exact current eligible count, and a typed interactive confirmation before delivery. Read
  Resend credentials and sender settings from the same `RESEND_*` environment variables as the
  app. Info logs contain counts; debug logs may show the
  candidate names and addresses. Generate links only immediately before sending, use Resend
  idempotency keys and at most four requests per second, persist a `0600` result ledger incrementally
  plus a secret-free resolved-settings manifest, and never overwrite either file. Verify the
  account's daily/monthly Resend quota first: the free transactional daily quota is below a
  150-recipient campaign. Keep open/click tracking disabled for these authentication links.
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
  - enabling and checking the implemented Firebase Admin SDK plus Resend path when custom
    delivery control is needed.
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
- For a reported `IP-XXXX-XXXX` support code, use Cloud Logging to filter the `authorization`
  component by `jsonPayload.authTrace` (and the `auth-diagnostics`, `send-magic-link`, and
  `firebase-email-link` components by the same field). The resulting timeline must show the
  route, required role, decision, and status for each traced protected API request. Treat the
  code as troubleshooting metadata, not a credential; it cannot authorize access by itself.
- If a member cannot see a support code while reporting a clearly stale interface, ask them to
  reload the site, then clear only `ipace-owners.org` website data if necessary. This signs them
  out; they must request a fresh magic link afterwards. Never ask them to send the magic link,
  a screenshot containing it, a Firebase token, or all of their browser data.

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
- Browser JavaScript has no transpilation step. ESLint must reject trailing commas in
  `src/assets/js/**` function calls and parameter lists, so ES2017-only comma syntax cannot pass
  CI. Scope that rule to browser functions: do not make Node scripts/tests, object literals, or
  array literals conform to an unnecessary legacy style.
- Keep tests for preview URL extraction, preview authorized-domain updates, and Function
  environment generation.
- Run `make lint`, `make build`, and `make test` after CI, deployment, or operational prompt
  changes.
