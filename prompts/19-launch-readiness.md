# Launch Readiness

Use this prompt when preparing the site for public launch, launch-week verification, or
post-launch triage. It consolidates launch-specific decisions that cut across public copy,
Join flow, authentication email delivery, infrastructure, privacy, and operational checks.

## Launch objective

The public launch site should ask owners to register interest and join the organised owner
group. It should not yet ask owners to submit detailed vehicle evidence. Evidence and vehicle
data collection remain available behind full mode and authenticated member tooling, but the
launch call to action is owner registration only.

The launch experience must communicate:

- I-PACE owners are working together for fair outcomes.
- The first focus is traction-battery problems, recalls/notices, and the need for a broadly
  acceptable resolution.
- The group is independent, organised, constructive, and not another discussion forum.
- JLR can hear owner priorities directly through an independent consent-based group.
- Registration does not enrol anyone in legal action; any future legal route would be
  explained separately and require a deliberate owner choice.
- Legal counsel exists as expert input and credible capability, not as the default posture.

## Public UX checks

Before launch:

- Verify launch mode is the default without relying on JavaScript to hide full-mode content.
- Check Home, About, FAQ, Join, Sign in, privacy, terms, and footer links on mobile,
  tablet, and desktop.
- Confirm the launch Join form asks only for name, email, contact consent, legal-claim
  acknowledgement, and privacy/participation links.
- Confirm completion copy says vehicle information can be added later and is optional.
- Confirm hidden full-mode public routes show a concise unavailable-at-launch panel when
  opened directly.
- Confirm member-protected functionality remains available after sign-in.
- Run keyboard and screen-reader spot checks for mobile navigation, form labels, focus
  order, and auth gates.

## Launch imagery

Use `public/images/ipace-hero.png` as the launch hero car image where appropriate. The image
is available for project use and should be referenced through `/images/ipace-hero.png` in
site templates and through an absolute custom-domain URL in email HTML.

Do not introduce Jaguar/JLR logos or copyrighted third-party imagery. Keep alt text neutral
and descriptive.

## Passwordless email delivery

Firebase remains the authentication authority. The site uses passwordless email-link sign-in.

Default path:

- If Resend is not configured, call Firebase/Identity Toolkit and allow Firebase to send the
  default passwordless email.
- Keep the Firebase public-facing project name set to `I-PACE Owners` in production so
  default emails do not show stale product names.
- Use a verified Firebase Hosting/Auth action domain for passwordless links. DNS cannot
  create an HTTP redirect; do not treat DNS CNAMEs as 302 redirects.

Branded Resend path:

- If both `RESEND_API_KEY` and `RESEND_FROM` are configured, generate the Firebase sign-in
  link server-side with Firebase Admin `EmailSignInLink`, then send the message through
  Resend.
- Include both HTML and plain-text bodies.
- Use `/images/ipace-hero.png` in the HTML email through an absolute HTTPS custom-domain
  asset URL.
- Never log raw action links, API keys, raw provider responses, request bodies, Firebase ID
  tokens, or personal email addresses.
- If Resend link generation or delivery fails, log a sanitized warning and fall back to
  Firebase default email delivery so the user still receives a sign-in link.

Required Resend configuration:

- `RESEND_API_KEY_<ENV>` as a GitHub environment secret. This may be created manually, or
  bootstrapped by supplying the sensitive OpenTofu `resend_api_key` variable. Leave
  `resend_api_key` empty when the GitHub secret is managed manually.
- `RESEND_FROM_<ENV>`, `RESEND_REPLY_TO_<ENV>`, and `RESEND_ASSET_BASE_URL_<ENV>` as GitHub
  environment variables, managed by OpenTofu where possible.
- Resend sender domain DNS must pass SPF/DKIM/DMARC checks before enabling production use.

## Infrastructure and data protection

Before production launch:

- Run production OpenTofu plan/apply and review outputs.
- Confirm Firestore production has point-in-time recovery, delete protection, Google-managed
  encryption at rest, and a scheduled backup policy. Staging does not need the production
  protection posture.
- Confirm Firebase Hosting custom domains are active and certificates are valid.
- Confirm Firebase Auth authorized domains include the canonical production domain and do
  not rely on stale PR preview domains.
- Confirm GitHub Actions production environment variables and secrets are populated.
- Confirm Cloud Functions are redeployed after environment changes that affect passwordless
  email delivery.

## Launch verification

Run:

- `make test`
- `make build`
- `tofu -chdir=infra/opentofu/env validate`
- production smoke tests against `https://ipace-owners.org`

Manually verify:

- owner registration from a fresh browser session;
- magic-link receipt, appearance, and click-through;
- fallback behaviour if Resend is not configured;
- signed-in account page load;
- mobile layouts at 320px, 375px, and 390px;
- no horizontal overflow;
- no full-mode content leakage in launch mode;
- footer legal/privacy links;
- social share metadata and preview cards.

## Launch rollback

Keep rollback simple:

- Resend can be disabled by removing/blanking `RESEND_API_KEY` or `RESEND_FROM` and
  redeploying Functions; Firebase default email delivery remains available.
- The launch/full presentation flag remains a presentation/discoverability control, not an
  authorization boundary.
- Changing the public default back from launch to full mode should require only the central
  site-mode configuration edit and deployment.
