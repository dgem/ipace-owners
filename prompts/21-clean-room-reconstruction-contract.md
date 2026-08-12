# Clean-room Reconstruction Contract

Use this terminal prompt as the final acceptance contract when recreating the repository from the
numbered prompts and `AGENTS.md`. It defines precedence, the minimum product surface, and the
information that must be preserved outside source control.

## Authority and precedence

- `00-original-project-prompt.md` is historical context only. Its Netlify architecture,
  placeholder data, and superseded product copy are not implementation requirements.
- Every file in `prompts/` must match
  `^\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$` (`xx-name.md`) and sequence numbers must remain
  contiguous.
- This contract must be the highest-numbered prompt. Add or renumber feature prompts before
  it, then renumber this file so reconstruction acceptance always runs after every feature
  contract.
- All numbered prompt files plus `AGENTS.md` describe the current product, except that
  prompt `00` remains historical context as noted above. Later, more specific prompts
  override earlier general prompts when requirements conflict.
- `09-architecture-overview.md` is authoritative for cross-layer architecture;
  feature-specific prompts are authoritative for their own request validation, copy, UX,
  storage, and authorization rules.
- A reconstruction is complete only when the acceptance checks below pass. Similar-looking
  pages without the documented security and data boundaries are not equivalent.

## Required repository and route surface

Recreate an Eleventy 3 project with the directory structure in `AGENTS.md`, a single custom
stylesheet, deferred vanilla-JavaScript IIFEs, Go Functions, Firebase Hosting configuration,
OpenTofu infrastructure, GitHub Actions, tests, scripts, prompts, and public assets.

The generated public route surface must include:

- `/`, `/about/`, `/faq/`, `/join/`, `/contact/`, `/privacy/`, `/terms/`,
  `/methodology/`, `/evidence-dashboard/`, and `/updates/`;
- dated or named update pages generated from `src/updates/`;
- `/member/dashboard/`, `/member/account/`, and `/member/submit-vehicle-data/`;
- `/admin/`, `/admin/review-queue/`, `/admin/outreach/`, `/admin/email-campaigns/`, and
  `/admin/instagram-campaigns/`;
- permanent redirects from `/account/**` and `/submit-vehicle-data/**` to their member
  equivalents;
- a generated 404 page, clean URLs, trailing slashes, and a final Hosting fallback to
  `/404.html`.

Public pages use the shared base/page/form layouts, header, mobile navigation, footer, SEO,
cards, callouts, authentication gate, and Join result partials. Private member/admin pages
must be `noindex, nofollow`. Canonical URLs, Open Graph metadata, Twitter metadata, and
Organisation/WebSite structured data belong in the shared SEO partial rather than being
duplicated per page.

The admin outreach route loads `outreach-assistant.js` only on that page. It generates explicit,
user-opened Facebook post-search URLs from editable search phrases and group URLs and drafts
editable issue-specific replies. It performs no Facebook network request, scraping, logged-in
session automation, automatic navigation, messaging, posting or Facebook-content persistence.

OpenTofu must reconcile the authoritative Firebase administrator email set through the Identity
Platform API because the Google provider has no Firebase Auth user data source. The shared module
always includes `dan@kanzi.co.uk`; additional admins come from environment configuration. Resolve
emails to per-environment UIDs, preserve unrelated claims, grant `admin: true`, remove only admin
access from removed users, and fail closed for missing accounts or an empty desired set.
Signed-in administrators receive desktop and mobile navigation to `/admin/` only after
the browser reads an admin claim from the Firebase ID token. Treat that link as discoverability,
not authorization; the route remains gated by the server-verified admin API.

Expose exactly one claim-gated Admin action in the primary desktop controls and one in the mobile
Member section. `/admin/` is the directory for individual tools; do not duplicate those tool links
in shared navigation. Member and admin routes render compact, single-line
`Member › current page` and `Admin › current page` secondary breadcrumbs with the active
crumb visibly emphasised. Admin tool section crumbs link back to `/admin/`, and page headings
do not repeat a redundant Admin eyebrow. The drawer provides My Data and Add Vehicle.
`My Data` links to `/member/account/`, and the member email is not displayed as a header
action. The signed-out mobile
header exposes Sign in beside the menu toggle and repeats it in the drawer for discoverability;
both signed-out actions disappear after authentication, and the mobile header action remains
hidden at desktop widths.

`/admin/` is the claim-gated landing dashboard. It links to every implemented admin tool and
describes planned areas without linking to unimplemented routes.

## API contract inventory

Firebase Hosting rewrites all `/api/**` traffic to one `Api` Function in `europe-west2`.
Recreate the method, authentication, validation, generic-error, ownership, and CORS
behaviour described in prompts `08` and `10-14` for these routes. Application-level request
rate limiting is not currently implemented; add it only as a deliberate, tested security
change rather than assuming it exists.

| Method and route | Authentication | Request/response contract |
|---|---|---|
| `POST /api/send-magic-link` | Public | JSON `email`, optional `name`; send only for a matching Join submission or existing Firebase Auth account and return enumeration-resistant `{ "ok": true }` for syntactically valid requests. |
| `POST /api/submit-join` | Optional Firebase token | `name`, `email`, `country`, `relationship`, `skills[]`, `consent-contact`, `consent-not-legal`, `consent-data`, and `bot-field`; save the Join record and initiate guest activation. |
| `POST /api/submit-vehicle-basics` | Member | `vin`, `registration`, `country`, `modelYear`, `mileage`, `ownedSince`, `firstReg`, plus optional `soh`, `sohDate`, `sohMileage`, `sohSource`. |
| `POST /api/submit-soh` | Member/vehicle owner | `vehicleId`, `soh`, `sohDate`, `sohMileage`, `sohSource`; append history and update the vehicle compatibility value. |
| `POST /api/upsert-service-event` | Member/vehicle and record owner | `id`, `vehicleId`, `eventType`, `occurredAt`, `mileage`, `title`, `description`, `status`, `campaigns[]`, `serviceProviderId`, `serviceProviderName`, `serviceProviderPostcode`, `serviceProviderAuthorised`, `finalFixAt`, `courtesyVehicleOffered`, `courtesyVehicleProvided`, `partsDelay`, `goodwillPayment`, `milesDrivenWhilstFaulty`, `warrantyCover`, `disputeStatus`. Derive `daysToFinalFix` server-side. |
| `GET /api/member-data` | Member | Return only that UID's private member snapshot. |
| `GET /api/member-export?format=csv\|xlsx` | Member | Return a private no-store ZIP of four CSV datasets or a formatted five-sheet Excel workbook built from only that UID's snapshot; omit internal IDs/hashes and neutralise spreadsheet formulas. |
| `GET /api/admin-data` | Admin claim | Return Join and vehicle review records. |
| `GET /api/admin/stats` | Admin claim | Return aggregate member, vehicle, SoH, and service-event statistics for the admin dashboard. |
| `POST /api/admin/reengagement-preview` | Admin claim | Return aggregate counts for consented Join submitters who have not registered. |
| `POST /api/admin/reengagement-send` | Admin claim | Require the campaign ID, exact eligible count and typed confirmation; recheck registrations and send the next batch of at most ten. |
| `POST /api/admin/member-referral-preview` | Admin claim | Preview aggregate counts and exact copy for registered accounts with matching contact consent. |
| `POST /api/admin/member-referral-send` | Admin claim | Confirm and send the next batch of at most ten referral emails with the same idempotent ledger safeguards. |
| `POST /api/admin/all-members-drive-preview` | Admin claim | Preview the deduplicated, contact-consenting audience across verified and unverified Join records and the exact recruitment email. |
| `POST /api/admin/all-members-drive-send` | Admin claim | Confirm and send the next batch of at most ten all-member recruitment emails with hashed idempotent delivery records. |
| `POST /api/admin/email-campaign-history` | Admin claim | Return parent campaign records and aggregate hashed-ledger delivery counts, including inferred legacy runs and cached Resend delivery outcomes, without addresses. |
| `POST /api/admin/custom-campaign-templates` | Admin claim | Return source-controlled custom-composer Markdown templates and metadata without sending or saving a campaign. |
| `POST /api/admin/custom-campaign-preview` | Admin claim | Validate/save a named subject and Markdown draft, calculate the verified consented audience, and return representative branded HTML/plain-text output. |
| `POST /api/admin/custom-campaign-send` | Admin claim | Load immutable saved content, recheck the audience and exact `SEND <count>` confirmation, then send at most ten idempotent messages. |
| `POST /api/admin/instagram-preview` | Admin claim | Validate a site-relative MP4/MOV path, caption and explicit full-media review; return the deterministic confirmation without a provider side effect. |
| `POST /api/admin/instagram-campaign-history` | Admin claim | List named drafts and immutable publication records, refreshing cached provider insights when available. |
| `POST /api/admin/campaign-summary` | Admin claim | Aggregate reconciled Resend email outcomes and Instagram publication/insight totals; report Facebook as manual unless Page Insights is connected. |
| `POST /api/admin/instagram-publish` | Admin claim | Revalidate the unchanged preview and exact confirmation, then create, process and publish one organic Reel through Meta. |
| `POST /api/admin/instagram-generate` | Admin claim | Reserve an idempotent job and start one billable eight-second 9:16 Veo operation after exact `GENERATE VIDEO` confirmation. |
| `POST /api/admin/instagram-generation-status` | Admin claim | Poll the Vertex operation, start the supported seven-second video continuation, promote the resulting 15-second master, and return an expiring delivery path. |
| `GET/HEAD /api/instagram-media/**` | Expiring bearer path | Validate a constant-time token hash and expiry, then range-stream the private master without exposing its GCS URI. |
| `GET /api/public-stats` | Public | Return the anonymised aggregate schema below with five-minute public caching and last-valid-snapshot fallback. |

The implemented API decoder accepts strict JSON bodies and rejects unknown fields. Shared
login forms still declare `method="POST"` so a JavaScript failure cannot leak email addresses
into a GET URL; their no-JavaScript POST is a safe failure, not a form-encoded API contract.
Reject invalid methods, malformed input, failed authentication, and unauthorized ownership;
return generic success for Join honeypot submissions as specified in the feature prompts.
Never depend on frontend gating for data protection.

## Canonical Firestore and snapshot schemas

Use these exact collection names: `joinSubmissions`, `members`, `vehicles`,
`batteryReadings`, `serviceEvents`, `memberSnapshots`, `emailCampaigns`, `campaignMetadata`,
`instagramCampaigns`, and `instagramGenerationJobs`. The email collection stores
campaign delivery subdocuments keyed by a non-reversible email fingerprint, with no recipient
address returned to the browser. Delivery documents may store a Resend provider ID, normalised
delivery status and provider update time; ignore provider recipient fields. `campaignMetadata`
caches the last Resend feedback reconciliation time and coverage state for five minutes.
Email campaign parent documents store custom/specialised campaign kind,
name, subject, Markdown, optional source run, lifecycle status, eligible/sent/failed/remaining
totals, batch count, and created/updated/last-sent timestamps. Private `members` documents may
store `emailVerifiedAt` and `emailVerifiedAtInferred`; do not expose either through public data.
`instagramCampaigns` stores named editable drafts and immutable processing, failed, or published
runs, optional source-run links, returned media IDs, and cached insight totals. It reserves the
saved ID before contacting Meta so a retry cannot silently duplicate a post.
`instagramGenerationJobs` stores prompt
hashes, phase, status, Vertex operation name, private object names, failure code, and only a hash
and expiry for each short-lived delivery token. Cloud Storage contains generated
snapshots under purpose-specific private/public object names; future evidence blobs require
their own validation and authorization design.

Every canonical evidence record contains `id`, `type`, `createdAt`, `updatedAt`, ownership
keys where applicable, and `{ status, verificationLevel }` review metadata. Preserve
`createdAt` on edits.

- Join records contain `identityUserId`, `userEmailHash`, contact `{ name, email, country }`,
  membership `{ relationship, skills[] }`, and consents `{ contact, notLegalClaim,
  anonymisedAnalysis }`.
- Vehicle records contain `identityUserId`, `userEmailHash`, vehicle `{ vinHash, vinLast6,
  registration, country, modelYear, mileage, ownedSince, firstRegistrationDate }`, battery
  `{ stateOfHealth, measuredAt, mileageAtMeasurement, source }`, and review metadata.
- Battery readings contain `identityUserId`, `vehicleId`, the battery object, and review
  metadata.
- Service events contain the fields listed in the API table, normalized numeric mileage and
  day counts, ownership keys, timestamps, and review metadata.
- Member snapshots contain `identityUserId`, `email`, `generatedAt`, `joinRecords[]`,
  `vehicleRecords[]`, `batteryReadings[]`, and `serviceEvents[]`.
- Public statistics currently use `schemaVersion: 6` and contain `generatedAt`, `joinedOwners`,
  `registeredMembers`,
  `ownersContributed`, `vehiclesRegistered`, `vehiclesWithSoh`, `sohReadings`,
  `serviceEventsLogged`,
  `vehiclesWithRepeatSoh`, optional `averageReportedSoh`, optional `averageSohChange`, and
  `{ label, count }[]` arrays for `sohDistribution` and `modelYearDistribution`.

Use Firestore timestamps for canonical timestamps and RFC 3339 JSON serialization. Optional
values must remain absent/null rather than becoming fabricated zero measurements. Full VINs
are never stored; use HMAC-SHA-256 with `VIN_PEPPER` and retain only the last six characters.
The public `joinedOwners` total comes from contact-consenting Join submissions deduplicated by
lowercased email with `+tag` aliases removed. The separate `registeredMembers` total comes from the
complete paginated Firebase Auth user list.

## Configuration contract

Build-time Firebase web configuration uses `FIREBASE_WEB_API_KEY`,
`FIREBASE_AUTH_DOMAIN`, `FIREBASE_PROJECT_ID`, `FIREBASE_APP_ID`, and
`FIREBASE_STORAGE_BUCKET`. Function runtime configuration uses `FIRESTORE_DATABASE_ID`,
`SNAPSHOT_BUCKET`, `VIN_PEPPER`, `ALLOWED_ORIGINS`, `FIREBASE_WEB_API_KEY`,
`FIREBASE_EMAIL_CONTINUE_URL`, optional `FIREBASE_EMAIL_LINK_DOMAIN`, and optional
`RESEND_API_KEY`, `RESEND_FROM`, `RESEND_REPLY_TO`, and `RESEND_ASSET_BASE_URL`.
Optional Instagram publishing additionally uses secret `INSTAGRAM_ACCESS_TOKEN` and non-secret
`INSTAGRAM_USER_ID`, `INSTAGRAM_GRAPH_API_VERSION`, and `INSTAGRAM_MEDIA_BASE_URL`. Absence or
invalidity must disable publishing while leaving local preview available.
OpenTofu creates the `instagram-access-token` Secret Manager container and grants the Function
runtime accessor permission, but must not create its secret version from a tfvars value. Only after
an operator adds a version should `instagram_publishing_enabled` expose the secret name and
non-secret account ID/version to the deployment workflow.
Asynchronous campaign-video generation uses non-secret `CAMPAIGN_MEDIA_BUCKET`, `VEO_LOCATION`,
and `VEO_MODEL_ID`, with `VEO_LOCATION` defaulting to the explicit Veo 3.1 processing region
`us-central1`. OpenTofu must enable `aiplatform.googleapis.com`, explicitly provision the managed
Vertex AI service identity, grant its intended service-agent role and bucket-scoped object access,
grant the Function runtime `roles/aiplatform.user` and object access only on its private
campaign-media bucket, expire only objects under `work/`, and retain/version approved objects under
`masters/`. The Function and private bucket remain in `europe-west2`, but Veo generation is not
UK-resident. Google runtime identity replaces API keys. Generation and Instagram publishing remain
separate confirmed actions. Provider failures must be recorded and shown to administrators through
safe classifications rather than arbitrary raw provider messages.

Never commit real values. Provide `.tfvars.example` files for staging and production and
derive non-secret GitHub environment variables from OpenTofu outputs. Production uses
`ipace-owners.org`, `auth.ipace-owners.org`, the production Firebase project/database, and
production-only Firestore delete protection, PITR, destroy prevention, and retained daily
backups. Staging uses its own project/database, `auth.stage.ipace-owners.org`, preview
channels, and deliberately reduced data-protection settings.

Firebase Hosting must reproduce the CSP and the `X-Frame-Options`,
`X-Content-Type-Options`, `Referrer-Policy`, and `Permissions-Policy` headers described by
the security prompts, plus immutable one-year caching for `/assets/**`. Passwordless login
forms explicitly use POST even when JavaScript intercepts them.

## Visual and content fidelity

- Preserve `docs/homepage-copy.md` as a portable Markdown rendering of the canonical homepage
  wording, with absolute production links and placeholders for live statistics.
- Provide an admin-only registration-reminder campaign page. It previews aggregate counts and
  the exact HTML email in a sandboxed frame, exposes the plain-text alternative on demand, and
  uses a safe link placeholder without exposing addresses. Keep the
  clearly labelled send controls visible but disabled until preview succeeds, require exact
  confirmation, recheck registration before sending, send bounded resumable batches, and
  persist a hashed idempotent delivery ledger.
- Render re-engagement, member-referral and all-member recruitment delivery from embedded Markdown Go templates into
  matching plain-text and escaped HTML bodies. Wrap HTML delivery in the shared magic-link email
  chrome with its compact text masthead, hero image, responsive table layout and pill-shaped
  primary and share actions; do not add a logo image. Keep the legally important
  contact-consent/unsubscribe footer outside the routinely edited Markdown body. Render supported
  Markdown emphasis as semantic italic text in HTML and omit its delimiters from plain text.
  Referral-style campaigns display a ready-to-copy “I-PACE owners are stronger together” post
  with an owner Join CTA and prefill the same text in platform composers where supported;
  LinkedIn and Instagram retain the visible copy because their web share flows cannot reliably
  prefill it.
- Provide a specialised all-joined-member recruitment tool that includes verified and unverified
  contact-consenting Join records deduped by canonical email. Preview the exact thanks/progress,
  formal-Jaguar-approach about members' shared concerns, a request for Jaguar to engage
  constructively on options for everyone, and the cited vehicle-population and sharing message before enabling the same
  confirmed, resumable ten-message delivery controls. Lead the subject with thanks for joining;
  thank recipients for their support and describe the 17 July launch as less than two weeks ago.
- Provide custom verified-member campaigns with server-validated Markdown, sandboxed branded HTML
  preview, plain-text preview, click-to-insert allowlisted substitutions, resumable confirmed
  batches, aggregate history, and clone-to-rerun behaviour. Present history before the
  new-campaign composer, include a direct create-new shortcut, and place safety guidance beside
  confirmation controls rather than in a persistent warning banner. Dedupe joined and verified
  member totals by canonical email. Support `membersJoined`, `membersVerified`, `memberFirstName`,
  `memberLastName`, the requested `memberTittle` spelling and `memberTitle` alias, `memberJoined`,
  `memberVerified`, private-member `memberVehicles` JSON, `vehiclesRegisteredCount`, and
  `vehiclesSoHReadingsCount`; reject arbitrary Go-template actions and unsafe link schemes.
- Reconcile hashed email delivery records against Resend's paginated sent-email API when an
  administrator refreshes campaign data. Cache checks for five minutes and surface delivered,
  awaiting-delivery, opened, clicked, delayed, bounced, suppressed, complained, provider-failed
  and combined undeliverable totals in campaign history and the Admin overview. Store no provider
  recipient addresses and degrade to cached feedback when Resend is temporarily unavailable.
  Reopen drafts and continue partial custom runs only with unchanged saved content; an edit after
  any delivery creates a new run linked to the original.
- Provide an admin-only Instagram campaign page following the same preview-before-side-effect
  interaction. Chat prepares the post; a human reviews the complete final media; the server
  validates the exact site-relative media path and caption; and an exact typed confirmation
  gates immediate organic Reel publishing. Show history before the workspace, reopen drafts,
  and clone published records for editing/reposting without mutating the original. Display cached
  provider insights when available. Put local email totals and Instagram publication/insight
  totals on the Admin home, while explicitly labelling Facebook as manual when Page Insights are
  not connected. Reconstruct its creative and safety contract from prompt `20`. Do not claim that
  paid ads, scheduling or automated engagement are implemented.

Prompts define visual intent, not the exact control points or pixels of generated artwork.
Therefore the following committed assets are preservation-critical and must be backed up with
the repository or an artifact archive:

- `public/favicon.png`;
- `public/images/ipace-hero.png`;
- `public/images/ipace-owners-logo.svg` and `public/images/ipace-owners-logo.png`;
- `public/images/ipace-owners-qr.svg`;
- `public/images/ipace-owners-card-front.svg` and
  `public/images/ipace-owners-card-front.png`;
- `public/images/ipace-owners-card-back.svg` and
  `public/images/ipace-owners-card-back.png`;
- `public/images/ipace-owners-card-front-hero.svg` and
  `public/images/ipace-owners-card-front-hero.png`;
- `public/images/ipace-owners-card-back-hero.svg` and
  `public/images/ipace-owners-card-back-hero.png`;

Do not preserve or reconstruct the rejected key-frame-composite launch Reel as an approved
asset. A native temporal Veo result becomes preservation-critical only after a human has watched
the complete video, approved its synchronized sound and controlled-stop portrayal, and explicitly
committed the approved export as `public/ipace-owners-instagram-launch-reel.mp4`.

Preserve `public/downloads/sample-ipace-owner-data.xlsx` as the public, fictional
demonstration of the member Excel export. Produce it with the production workbook builder,
include no real member data, retain all five export sheets and native charts, and keep it
linked from both the member account and its launch Updates post.

The live privacy notice is not a placeholder. Keep it aligned with collected data, purposes
and lawful bases, authentication, processors and transfers, retention, cookies,
authenticated exports, data-subject rights, consent withdrawal, and the ICO complaint
route. Material expansion of collection or a change in organisational status still
requires human legal/privacy review.

A printable PDF is not currently committed and must not be presented as a recoverable source
artifact. Do not assume an image model can recreate the approved assets identically from
prompt text.

If those assets are genuinely unavailable, regenerate them from prompt `19`, label the
result as a new visual revision, verify QR scanning and print dimensions, and obtain human
design approval. Never silently claim pixel equivalence. Recreate page copy from prompts
`04`, `18`, and `19`; preserve British English, constructive advocacy tone, official launch
date of 17 July 2026, the canonicalised Join-submission “Owners joined” count, and the reusable
responsive gold racing-wreath presentation shared by the launch-page owners, registered-cars,
SoH-readings, and service/fault-record counters. Keep the three evidence counters within the
hero as a transparent subsection over the same graduated background rather than introducing
a separate solid-colour band. Place them directly beneath the hero image, retaining that
image-column composition on desktop so the following `Why now?` section is not needlessly
pushed below the fold. Keep all three on one row at mobile and desktop widths and make them
clearly smaller than the headline Owners joined wreath. Show authenticated members a
restrained, evidence-led CTA from those counters to the member vehicle-data workspace, using
spacing rather than horizontal rules to separate the subsection and CTA. The complete
evidence subsection, including the three counters, heading and CTA, is authenticated-member
UI and remains hidden for signed-out visitors. The complete
signed-in CTA must remain fully contained and reachable when scrolling the hero at a current
Android-phone responsive breakpoint, without being clipped or overflowing the viewport width.
Keep the nudge text and action together as one compact vertical subsection at every width,
with centred text before a deliberately small centred button rather than a full-width action.
Include a 490 × 874 Firefox responsive-design checkpoint so the intermediate mobile breakpoint
cannot regress into a clipped horizontal layout.

## Dependency, infrastructure, and CI contract

- Use the current production-supported versions mandated by `AGENTS.md`, then generate and
  commit npm, Go, and OpenTofu lock/checksum files. Exact historical dependency bytes are not
  part of a clean-room rebuild unless archived lockfiles are supplied. Preserve narrow,
  pinned npm transitive-security overrides while current direct tools still require affected
  ranges; verify them with a zero-result npm audit and the affected CLI/build paths, then
  remove them once upstream dependency ranges include the patched releases.
- Recreate staging and production workflows, read-only PR validation, repository-owner plus
  same-repository preview deployment gating, approval for all external contributors,
  job-scoped permissions, immutable third-party Action pins, Workload Identity Federation,
  serialized staging and production deployment, runtime-authorized public snapshot
  regeneration verified by smoke testing, branch-run cancellation, direct smoke testing,
  weekly Dependabot coverage, CodeQL `security-extended`, dependency review, npm audit,
  pinned `govulncheck`, and the passive ZAP baseline described in prompt `17`.
- Recreate the shared OpenTofu module, environment workspaces, Firebase/Identity Platform,
  named Firestore database, Storage bucket, runtime service account, least-privilege GitHub
  deployer, custom domains, DNS outputs, Secret Manager entries, GitHub environments, and
  optional Resend domain/configuration. Manual Fasthosts DNS remains an external operation.

## Clean-room acceptance checklist

Before declaring reconstruction complete:

1. Run `make install`, `make lint`, `make test`, `make build`, `make audit`, and OpenTofu
   formatting/validation for the environment root.
2. Confirm every route and redirect above, mobile navigation in signed-out/signed-in/admin
   states, launch/full site modes, canonical metadata, keyboard flows, and WCAG AA
   colour/focus behaviour.
3. Test Join and magic-link flows for new, existing, unknown, malformed, and honeypot users;
   confirm generic responses and that email addresses never enter fallback page URLs.
4. Test unauthenticated, member, wrong-owner, and admin authorization for every private API.
5. Test multiple vehicles, SoH history, service-event create/edit, snapshot regeneration,
   Firebase Auth pagination, five-minute public caching, and snapshot fallback.
6. Test registration-reminder, member-referral, and custom email campaign preview/send
   boundaries, immutable delivery ledgers, aggregate history, partial-run continuation,
   draft editing, clone-to-rerun behaviour, substitutions, exact confirmation gates, and the
   compact history/tools navigation plus keyboard-operable campaign tabs. At narrow
   breakpoints, show the four campaign tabs in a fully visible two-column grid rather than
   clipping later options behind an unlabelled horizontal overflow edge.
7. Test Instagram generation and publishing independently: admin/origin authorization,
   idempotent asynchronous Veo phases, private expiring media delivery, complete human review,
   preview invalidation, fail-closed optional configuration, and exact publish confirmation.
   Confirm generation completion can never trigger publication.
8. Confirm no raw VIN, token, email, name, private snapshot, or evidence record appears in
   public files, logs, aggregate responses, or static Hosting objects.
9. Plan staging infrastructure, deploy a PR preview, authorize only its scoped Firebase host,
   run smoke and passive ZAP checks, and retain reports as CI artifacts.
10. Visually compare desktop and mobile pages with approved references; scan the QR code and
   render both business-card sides at print size.
11. Obtain human review for logic, security, accessibility, legal/privacy copy, and tone before
   production deployment.

## Reproducibility verification strategy

CI must render deterministic credential-free admin states in Chrome at desktop and mobile
viewports, assert header expansion, menu placement, visible/disabled campaign controls and no
horizontal overflow, and upload the screenshots for human inspection. Any PR changing layout,
navigation, responsive behaviour, gating, or major page composition requires these checkpoints.

Use layered verification rather than claiming that an in-place test proves a clean-room
rebuild:

- Run `test/reconstruction-contract.test.js` in every `make test`. It checks that the
  maintained prompt range, required source routes, Hosting redirects/fallback, API router,
  implemented Firestore collections, runtime configuration names, and preservation-critical
  assets remain represented by this contract. It also verifies that documented `make`
  commands still exist.
- Treat this contract test as a drift detector, not proof that an empty machine can deploy
  the product. It deliberately does not exercise cloud credentials, live data restoration,
  DNS, email delivery, or visual equivalence.
- Periodically perform a true isolated reconstruction in a new empty repository or ephemeral
  environment using only `AGENTS.md`, prompts `01-21`, approved public assets, and separately
  supplied secrets/configuration. Run the full acceptance checklist and compare the resulting
  route/API/schema inventories with production before calling the reconstruction successful.
- Record the source commit, toolchain versions, generated lockfiles, deviations, elapsed
  time, and any undocumented operator knowledge discovered by that exercise; immediately
  externalise the findings into the maintained prompts.

## External backup and disaster-recovery boundary

Prompts cannot recreate live state. Maintain separately secured, tested backups or exports
for Firebase Auth users, Firestore data, Storage objects, DNS records, GitHub environment
variables/secrets, Secret Manager versions, deployed project identifiers, custom claims,
Resend configuration, and approved visual binaries. Record restoration order and responsible
owners in the private operational runbook. Never place those exports or secret values in this
public repository.

A prompt-only rebuild without those backups creates a new empty deployment. It is a product
reconstruction, not production disaster recovery.
