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
  acknowledgement, optional anonymised aggregate-publication consent, and
  privacy/participation links.
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

### Owners' group logo

Keep the generated logo source at `public/images/ipace-owners-logo.png`. To recreate or refine
it with an image-generation model, use this prompt as the baseline:

> Create a clean, vector-friendly side-profile logo of an I-PACE-style electric crossover in
> strict side view, facing left. Use a solid black vehicle body and tyres, with all windows and
> glass rendered as pure white negative space. Avoid manufacturer badges, Jaguar wordmarks, and
> other third-party marks. Set `iPace-Owners.org` as a friendly rounded italic sans-serif racing
> decal, using one consistent font size and weight from beginning to end. Run the decal diagonally
> upward from just behind the front wheel towards the rear side window. Use a pure white background,
> crisp flat shapes, no greys, gradients, shadows, reflections, extra text, or watermark. Keep the
> silhouette bold and suitable for later vector tracing.

The standalone logo and business-card front must use the same car-and-decal artwork. Set the decal
in bright teal (`#2DD4BF`), keep its typography and angle identical in both assets, and end the URL
around the rear-wheel centreline so `.org` retains normal spacing and does not appear compressed.
When either treatment changes, regenerate the other from the same source artwork rather than
allowing the logo and card to drift independently.

Use `public/favicon.png` as the favicon-sized logo treatment. Derive it directly from the same
black side-profile car used by the logo, remove the URL decal completely, and scale the full car
proportionally onto a square white canvas. Do not substitute a separately drawn or stylised car.

### Printable owner-distribution cards

Store the generated card artwork at:

- `public/images/ipace-owners-card-front.svg`
- `public/images/ipace-owners-card-front.png`
- `public/images/ipace-owners-card-back.svg`
- `public/images/ipace-owners-card-back.png`
- `public/images/ipace-owners-card-front-hero.svg`
- `public/images/ipace-owners-card-front-hero.png`
- `public/images/ipace-owners-card-back-hero.svg`
- `public/images/ipace-owners-card-back-hero.png`

The original front/back pair uses a straight-on UK landscape business-card composition,
approximately 85 × 55 mm, with a white background, a thin dark-teal (`#0f766e`) rounded
border, generous print-safe margins, friendly rounded sans-serif typography, and flat colours
only. The photographic pair is the documented full-bleed exception below. Do not render the
cards as physical mockups and do not add phone numbers, manufacturer badges, or watermarks.

Generate the front from the logo with this prompt:

> Create the front of a landscape advocacy business card. At the top, set
> `H570 Battery issues?` on one line in a bold, friendly italic rounded sans-serif, leaving
> enough space at the top right for a functional QR code on the same visual line.
> Place the black side-view car logo across the centre. Preserve its white glass and wheel details,
> and set the diagonal `iPace-Owners.org` decal in vivid bright teal (`#2DD4BF`). Across the bottom,
> use a solid dark-teal (`#0f766e`) rounded CTA band with centred white text reading exactly
> `Join us to help get a fair deal for all.` Keep every element inside print-safe margins and make
> the copy readable at physical business-card size.

Generate the front QR code deterministically rather than asking an image model to draw it. Encode
`https://ipace-owners.org` with high error correction, retain the full four-module white quiet zone,
and place it at the top right in line with the headline. Target at least 10 mm square at final print
size and test the printed proof with multiple phone cameras before distribution. Never distort,
rotate, crop, recolour, or place artwork inside the QR modules.

Keep an alternative photographic front using the website hero image. Compose it
deterministically in SVG rather than asking an image model to redraw the vehicle or critical
text. Crop the hero to the same 85 × 55 mm landscape proportion, use a restrained navy overlay
for legibility, but keep the photograph square-cornered and full bleed rather than clipping it to
a rounded card shape. Shift the crop left enough to preserve breathing room between the rear of
the car and the right edge. Omit a separate `I-PACE OWNERS` label and stack all white copy at top left: the
exact italic `H570 battery issues?` headline, `Join us to help get a fair deal for all.`,
`Free to join, takes less than a min.`, and `iPace-Owners.org`. Do not use a pill, badge,
separate lower background panel, or outer rounded white card frame. Remove the QR's white card,
then align the QR at bottom left with the copy column, using white modules over the dark road. Keep
it visually separate from the vehicle while preserving its full quiet-zone spacing and at least
10 mm final printed size. Embed the hero PNG as a data URI. Embed a dedicated QR SVG data URI with
its white background removed, its modules rendered white, and slight translucency applied to the
whole QR; do not rely on a filter that can make the background reappear in some SVG renderers.
The finished SVG must be one portable file with no sibling-asset dependency. Also commit a
self-contained, print-proportioned PNG render.

Generate the matching back with this prompt:

> Create the back of the same landscape advocacy business card, matching the front's white
> background, thin dark-teal rounded border, friendly typography, spacing, and teal accents. Do not
> repeat the car. Use the headline `I-PACE owners working together for fair outcomes`, followed by
> `An independent owners' group engaging constructively with JLR to pursue a fair and consistent
> resolution.` Add a `Why now?` section in a tidy two-column grid with four items:
> `Recall campaigns — H57x recall and customer-notice series`; `Traction battery faults — Power loss, warnings
> reduced range and performance`; `Battery warranty pressure — 8 years / 100,000 miles`; and
> `Inconsistent support — Delays, loan cars, goodwill and warranty outcomes`. Finish with a compact
> dark-teal strip reading `Free to join • Less than a minute • iPace-Owners.org`. Use accessible
> contrast and card-size-readable text; do not invent statistics or legal claims.

Pair the photographic front with its own photographic back rather than the white infographic
back. Reuse the square-cornered, full-bleed hero crop under a strong navy wash and embed the hero
PNG as a data URI so the SVG remains portable as one file. Set
`I-PACE owners working together for fair outcomes` as one compact white line, followed by
`Engaging constructively with Jaguar.`
Keep the content to three
card-size-readable points: `H57x recall series — Owners facing related battery recall and
customer-notice work`; `Traction battery faults — Power loss, warnings, reduced range and performance`;
and `Fair, consistent support — Warranty, repairs, loan cars and goodwill`. Finish with
`Independent • Evidence-led • Constructive`. Do not repeat the QR code or crowd the card with the
complete notice list.

Where a complete recall/customer-notice inventory is required in structured member data, use
H441, H448, H570, H571 and H572. Public card copy may refer to the H57x series when the card's
purpose is specifically H570 recruitment.

Treat generated text as artwork that requires proofreading. Before publishing or printing, verify
every campaign number, URL, apostrophe, slash, capital letter, and punctuation mark against the
copy above. Preserve editable or higher-resolution source artwork when preparing a commercial
print run; the committed PNG files are the approved visual references.

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
- Use `/images/ipace-hero.png` in the HTML email through an absolute HTTPS asset URL. In PR
  previews, derive the asset URL from the PR preview origin so the email references the
  image deployed on the same preview branch. In normal staging/production, use the configured
  stable asset base URL. Use the canonical `https://ipace-owners.org` asset base for staging
  delivery because the `stage.ipace-owners.org` sender domain does not serve Hosting assets.
- Never log raw action links, API keys, raw provider responses, request bodies, Firebase ID
  tokens, or personal email addresses.
- If Resend link generation or delivery fails, log a sanitized warning and fall back to
  Firebase default email delivery so the user still receives a sign-in link.

Required Resend configuration:

- `RESEND_API_KEY_<ENV>` as a GitHub environment secret. This may be created manually, or
  bootstrapped by setting `bootstrap_resend_api_key_secret = true` and supplying the
  sensitive OpenTofu `resend_api_key` variable. Leave the bootstrap boolean false when the
  GitHub secret is managed manually.
- `RESEND_FROM_<ENV>`, `RESEND_REPLY_TO_<ENV>`, and `RESEND_ASSET_BASE_URL_<ENV>` as GitHub
  environment variables, managed by OpenTofu where possible.
- The Resend sending domain may be created/read by OpenTofu with
  `manage_resend_domain = true` and `resend_api_key` supplied through an uncommitted tfvars
  file or `TF_VAR_resend_api_key`. Use `make infra-resend-dns-records ENV=<environment>` to
  print the required Resend DNS records, then add them at Fasthosts while Fasthosts remains
  authoritative. Resend sender domain DNS must pass SPF/DKIM/DMARC checks before enabling
  production use.

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
