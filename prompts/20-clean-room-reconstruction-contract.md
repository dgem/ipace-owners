# Clean-room Reconstruction Contract

Use this prompt as the final acceptance contract when recreating the repository from the
numbered prompts and `AGENTS.md`. It defines precedence, the minimum product surface, and the
information that must be preserved outside source control.

## Authority and precedence

- `00-original-project-prompt.md` is historical context only. Its Netlify architecture,
  placeholder data, and superseded product copy are not implementation requirements.
- Every file in `prompts/` must match
  `^\d{2}-[a-z0-9]+(?:-[a-z0-9]+)*\.md$` (`xx-name.md`) and sequence numbers must remain
  contiguous.
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
- `/admin/`, `/admin/review-queue/`, `/admin/outreach/`, and `/admin/email-campaigns/`;
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
Signed-in administrators receive desktop and mobile navigation to `/admin/outreach/` only after
the browser reads an admin claim from the Firebase ID token. Treat that link as discoverability,
not authorization; the route remains gated by the server-verified admin API.

The complete admin menu belongs in a claim-gated, right-aligned secondary desktop header row and
a labelled mobile-drawer section, not inside individual admin page content.

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
| `POST /api/send-magic-link` | Public | JSON `email`, optional `name`; return enumeration-resistant `{ "ok": true }` for syntactically valid requests. |
| `POST /api/submit-join` | Optional Firebase token | `name`, `email`, `country`, `relationship`, `skills[]`, `consent-contact`, `consent-not-legal`, `consent-data`, and `bot-field`; save the Join record and initiate guest activation. |
| `POST /api/submit-vehicle-basics` | Member | `vin`, `registration`, `country`, `modelYear`, `mileage`, `ownedSince`, `firstReg`, plus optional `soh`, `sohDate`, `sohMileage`, `sohSource`. |
| `POST /api/submit-soh` | Member/vehicle owner | `vehicleId`, `soh`, `sohDate`, `sohMileage`, `sohSource`; append history and update the vehicle compatibility value. |
| `POST /api/upsert-service-event` | Member/vehicle and record owner | `id`, `vehicleId`, `eventType`, `occurredAt`, `mileage`, `title`, `description`, `status`, `campaigns[]`, `finalFixAt`, `daysToFinalFix`, `courtesyVehicleOffered`, `courtesyVehicleProvided`, `partsDelay`, `warrantyCover`, `disputeStatus`. |
| `GET /api/member-data` | Member | Return only that UID's private member snapshot. |
| `GET /api/admin-data` | Admin claim | Return Join and vehicle review records. |
| `POST /api/admin/reengagement-preview` | Admin claim | Return aggregate counts for consented Join submitters who have not registered. |
| `POST /api/admin/reengagement-send` | Admin claim | Require the campaign ID, exact eligible count and typed confirmation; recheck registrations and send the next batch of at most ten. |
| `POST /api/admin/member-referral-preview` | Admin claim | Preview aggregate counts and exact copy for registered accounts with matching contact consent. |
| `POST /api/admin/member-referral-send` | Admin claim | Confirm and send the next batch of at most ten referral emails with the same idempotent ledger safeguards. |
| `GET /api/public-stats` | Public | Return the anonymised aggregate schema below with five-minute public caching and last-valid-snapshot fallback. |

The implemented API decoder accepts strict JSON bodies and rejects unknown fields. Shared
login forms still declare `method="POST"` so a JavaScript failure cannot leak email addresses
into a GET URL; their no-JavaScript POST is a safe failure, not a form-encoded API contract.
Reject invalid methods, malformed input, failed authentication, and unauthorized ownership;
return generic success for Join honeypot submissions as specified in the feature prompts.
Never depend on frontend gating for data protection.

## Canonical Firestore and snapshot schemas

Use these exact collection names: `joinSubmissions`, `members`, `vehicles`,
`batteryReadings`, `serviceEvents`, `memberSnapshots`, and `emailCampaigns`. The latter stores
campaign delivery subdocuments keyed by a non-reversible email fingerprint, with no recipient
address returned to the browser. Cloud Storage contains generated
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
- Public statistics currently use `schemaVersion: 5` and contain `generatedAt`, `joinedOwners`,
  `registeredMembers`,
  `ownersContributed`, `vehiclesRegistered`, `vehiclesWithSoh`, `sohReadings`,
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
  the exact plain-text email with a safe link placeholder without exposing addresses. Keep the
  clearly labelled send controls visible but disabled until preview succeeds, require exact
  confirmation, recheck registration before sending, send bounded resumable batches, and
  persist a hashed idempotent delivery ledger.

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
  `public/images/ipace-owners-card-back-hero.png`.

A printable PDF is not currently committed and must not be presented as a recoverable source
artifact. Do not assume an image model can recreate the approved assets identically from
prompt text.

If those assets are genuinely unavailable, regenerate them from prompt `19`, label the
result as a new visual revision, verify QR scanning and print dimensions, and obtain human
design approval. Never silently claim pixel equivalence. Recreate page copy from prompts
`04`, `18`, and `19`; preserve British English, constructive advocacy tone, official launch
date of 17 July 2026, the canonicalised Join-submission “Owners joined” count, and its responsive garland
presentation.

## Dependency, infrastructure, and CI contract

- Use the current production-supported versions mandated by `AGENTS.md`, then generate and
  commit npm, Go, and OpenTofu lock/checksum files. Exact historical dependency bytes are not
  part of a clean-room rebuild unless archived lockfiles are supplied.
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
2. Confirm every route and redirect above, mobile navigation, launch/full site modes,
   canonical metadata, keyboard flows, and WCAG AA colour/focus behaviour.
3. Test Join and magic-link flows for new, existing, unknown, malformed, and honeypot users;
   confirm generic responses and that email addresses never enter fallback page URLs.
4. Test unauthenticated, member, wrong-owner, and admin authorization for every private API.
5. Test multiple vehicles, SoH history, service-event create/edit, snapshot regeneration,
   Firebase Auth pagination, five-minute public caching, and snapshot fallback.
6. Confirm no raw VIN, token, email, name, private snapshot, or evidence record appears in
   public files, logs, aggregate responses, or static Hosting objects.
7. Plan staging infrastructure, deploy a PR preview, authorize only its scoped Firebase host,
   run smoke and passive ZAP checks, and retain reports as CI artifacts.
8. Visually compare desktop and mobile pages with approved references; scan the QR code and
   render both business-card sides at print size.
9. Obtain human review for logic, security, accessibility, legal/privacy copy, and tone before
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
  environment using only `AGENTS.md`, prompts `01-20`, approved public assets, and separately
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
