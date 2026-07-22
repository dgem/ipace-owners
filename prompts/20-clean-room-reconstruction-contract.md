# Clean-room Reconstruction Contract

Use this prompt as the final acceptance contract when recreating the repository from the
numbered prompts and `AGENTS.md`. It defines precedence, the minimum product surface, and the
information that must be preserved outside source control.

## Authority and precedence

- `00-original-project-prompt.md` is historical context only. Its Netlify architecture,
  placeholder data, and superseded product copy are not implementation requirements.
- `01-20` plus `AGENTS.md` describe the current product. Later, more specific prompts
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
- `/admin/review-queue/`;
- permanent redirects from `/account/**` and `/submit-vehicle-data/**` to their member
  equivalents;
- a generated 404 page, clean URLs, trailing slashes, and a final Hosting fallback to
  `/404.html`.

Public pages use the shared base/page/form layouts, header, mobile navigation, footer, SEO,
cards, callouts, authentication gate, and Join result partials. Private member/admin pages
must be `noindex, nofollow`. Canonical URLs, Open Graph metadata, Twitter metadata, and
Organisation/WebSite structured data belong in the shared SEO partial rather than being
duplicated per page.

## API contract inventory

Firebase Hosting rewrites all `/api/**` traffic to one `Api` Function in `europe-west2`.
Recreate the method, authentication, validation, generic-error, ownership, CORS, and
rate-limiting behaviour described in prompts `08` and `10-14` for these routes:

| Method and route | Authentication | Request/response contract |
|---|---|---|
| `POST /api/send-magic-link` | Public | JSON/form `email`, optional `name`; return enumeration-resistant `{ "ok": true }` for syntactically valid requests. |
| `POST /api/submit-join` | Optional Firebase token | `name`, `email`, `country`, `relationship`, `skills[]`, `consent-contact`, `consent-not-legal`, `consent-data`, and `bot-field`; save the Join record and initiate guest activation. |
| `POST /api/submit-vehicle-basics` | Member | `vin`, `registration`, `country`, `modelYear`, `mileage`, `ownedSince`, `firstReg`, plus optional `soh`, `sohDate`, `sohMileage`, `sohSource`. |
| `POST /api/submit-soh` | Member/vehicle owner | `vehicleId`, `soh`, `sohDate`, `sohMileage`, `sohSource`; append history and update the vehicle compatibility value. |
| `POST /api/upsert-service-event` | Member/vehicle and record owner | `id`, `vehicleId`, `eventType`, `occurredAt`, `mileage`, `title`, `description`, `status`, `campaigns[]`, `finalFixAt`, `daysToFinalFix`, `courtesyVehicleOffered`, `courtesyVehicleProvided`, `partsDelay`, `warrantyCover`, `disputeStatus`. |
| `GET /api/member-data` | Member | Return only that UID's private member snapshot. |
| `GET /api/admin-data` | Admin claim | Return Join and vehicle review records. |
| `GET /api/public-stats` | Public | Return the anonymised aggregate schema below with five-minute public caching and last-valid-snapshot fallback. |

Accept JSON and browser form encoding where the UI uses either. Reject invalid methods,
malformed input, failed authentication, unauthorized ownership, and honeypot submissions as
specified in the feature prompts. Never depend on frontend gating for data protection.

## Canonical Firestore and snapshot schemas

Use these exact collection names: `joinSubmissions`, `members`, `vehicles`,
`batteryReadings`, `serviceEvents`, and `memberSnapshots`. Cloud Storage contains generated
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
- Public statistics contain `schemaVersion`, `generatedAt`, `joinedOwners`, `registeredMembers`,
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

Prompts define visual intent, not the exact control points or pixels of generated artwork.
Therefore the following are preservation-critical source assets and must be backed up with
the repository or an artifact archive: logo SVG/PNG, favicon, hero image, QR SVG, business
card front/back SVG/PNG, and printable business-card PDF. Do not assume an image model can
recreate them identically from prompt text.

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
- Recreate staging and production workflows, job-scoped permissions, Workload Identity
  Federation, serialized preview deployment, branch-run cancellation, direct smoke testing,
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

## External backup and disaster-recovery boundary

Prompts cannot recreate live state. Maintain separately secured, tested backups or exports
for Firebase Auth users, Firestore data, Storage objects, DNS records, GitHub environment
variables/secrets, Secret Manager versions, deployed project identifiers, custom claims,
Resend configuration, and approved visual binaries. Record restoration order and responsible
owners in the private operational runbook. Never place those exports or secret values in this
public repository.

A prompt-only rebuild without those backups creates a new empty deployment. It is a product
reconstruction, not production disaster recovery.
