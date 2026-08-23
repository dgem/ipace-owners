# Member Vehicle Workspace and Service History Prompt

Use this prompt when changing the signed-in member dashboard, SoH history presentation, or
service/fault records.

## Goal

Give members a full-width working view of each registered I-PACE. Support multiple cars
without compressing each car into a narrow dashboard column.

## Vehicle Navigation

- Render one selected vehicle at a time on desktop and mobile.
- Put registered vehicles in an accessible tab list above the workspace.
- Support Left and Right Arrow navigation between tabs.
- Keep Add vehicle as a separate button linking to the vehicle-basics form.
- Use registration as the tab label when available, otherwise a non-sensitive generated
  vehicle reference.
- Keep membership details and group updates below the vehicle workspace as secondary panels.

## State of Health History

- Show all dated SoH readings for the selected vehicle in an accessible SVG line graph.
- Include a readable table so exact values, dates, mileage, and sources do not depend on
  interpreting the graph.
- Provide an Add reading button that reveals a focused form.
- Post to `POST /api/submit-soh` with a Firebase ID token.
- Refresh private member data after a successful write while preserving the selected car.
- SoH measurement dates must not be in the future. Enforce this in the browser and in the
  Go handler.

## Service Events and Faults

Below SoH history, show a dated timeline for the selected car. Support these record types:

- service;
- fault;
- repair;
- recall;
- inspection;
- other.

Each record contains date, optional mileage, summary, optional details, and status (`open`,
`monitoring`, `resolved`, or `completed`). Members can add and edit records through
`POST /api/upsert-service-event`.
Service/fault event dates must not be in the future. Enforce this in the browser and in the
Go handler.

New records should default to `fault`. The form should also capture optional structured
evidence fields:

- a single service-provider lookup searchable by provider name, town, postcode, county, or
  address fragment, backed by a
  refreshable static snapshot of Jaguar UK's official Electric Vehicle Service directory;
  store the locator CI code, name, postcode, and member-confirmed authorised-JLR status;
  allow a member to enter a provider not present in the suggestions;
- related campaigns or recalls: `H441`, `H448`, `H570`, `H571`, `H572`, other, unsure,
  none, presented as a compact wrapping grid that remains overflow-safe at every width;
- final fix date; calculate days from fault to final fix on the server rather than accepting
  a member-entered duration, and show the same calculation in the browser as immediate help;
- whether a courtesy vehicle was offered and whether one was provided;
- the parts-delay range: none, up to one week, up to one month, up to two months, up to
  three months, or four months and over;
- whether a goodwill payment was received and optional miles driven whilst faulty;
- warranty cover in place at the time;
- responsibility or warranty dispute status.

## Local service-document importer

Provide authenticated members with `/member/import-service-records/`, linked from the
vehicle workspace. It is a private, browser-only assistant for turning local PDFs and
JPEG/PNG/WebP scans into reviewed service-event drafts. Do not upload or persist the source
files. Use PDF.js to extract embedded text from native PDFs; where that is insufficient,
render every PDF page or use the supplied image and run Tesseract WebAssembly OCR locally.
Ship the PDF, OCR worker/core, and English trained-data assets with the static site, rather
than calling a third-party document or OCR service.

Hash each selected file using SHA-256 in the browser and suppress duplicates for the current
session. Use filename/text hints only to draft date, mileage, invoice category, type, status,
provider, and a short title. Show the source filename and local fingerprint alongside editable
vehicle, date, mileage, provider, type, status, title, and details fields, and tell members to
keep the original document open while reviewing. Never submit automatically;
the member must use an explicit `Submit reviewed record` control. Submit only the reviewed
structured JSON to the existing authenticated `POST /api/upsert-service-event`. Keep the source
filename, digest, and raw OCR output local; do not seed any of them into submitted fields because
they may contain personal data. The original document remains local.

Refresh `src/assets/data/jaguar-uk-service-providers.json` with
`make update-service-providers`. The generator queries the official Jaguar UK retailer
locator's Electric Vehicle Service and Electric Vehicle Battery Repair filters, retains
source/retrieval metadata, and rejects implausibly small results. This is a suggestion list,
not a warranty of current capability; the UI must tell members to confirm with the provider.

The Go Function must verify Firebase Auth, vehicle ownership, and existing-record ownership.
Store canonical records in Firestore `serviceEvents`, preserve creation/review metadata on
edits, and regenerate the private member snapshot after every successful write. Do not add
these records to public statistics until consent, moderation, and publication rules exist.

## Member data export

The Account page provides two authenticated downloads through
`GET /api/member-export?format=csv|xlsx`:

- `csv` is a ZIP containing separate `membership.csv`, `vehicles.csv`,
  `soh-readings.csv`, and `service-and-fault-history.csv` files.
- `xlsx` is a professionally formatted workbook with Summary, Membership, Vehicles,
  SoH History, and Service & Faults sheets. Add native SoH and event-type charts only when
  source rows exist.

Build both formats server-side from the authenticated member snapshot. Send the Firebase ID
token only in the Authorization header, return private no-store attachment responses, omit
internal identity IDs and email/VIN hashes, expose only the retained VIN final six characters,
and neutralise text that spreadsheet software could interpret as a formula. Empty datasets
remain valid exports. The browser should announce preparation, success, and failure states.

Commit a public sample workbook at
`public/downloads/sample-ipace-owner-data.xlsx`. Generate it through the production workbook
builder using fictional records only, including `.test` email addresses. Link it from the
member account export controls and the launch Updates post. The sample must retain the
production workbook structure: Summary, Membership, Vehicles, SoH History, and Service &
Faults sheets, plus native summary charts. Automated tests must verify its fictional
identity, sheet structure, representative totals, and chart parts.

## Tests

- Test unauthenticated rejection and input validation.
- Test that the document importer is member-gated, uses local PDF/OCR assets, hashes files,
  OCRs every scanned PDF page, continues after an unreadable file, includes a review-and-submit
  boundary, and has no document-upload endpoint or filename/raw-OCR leakage in its payload.
- Test ownership predicates for both user and vehicle IDs.
- Test Firebase Hosting route and browser bearer-token wiring.
- Test tab semantics, full-width workspace markup, graph accessibility, and add/edit controls.
- Test the provider-directory shape, server-derived resolution duration, new service fields,
  and its accessible in-page provider suggestions, which must match provider name, town,
  postcode, county, and address fragments. Campaign choices must wrap in a compact grid at
  desktop and mobile widths without creating horizontal form overflow.
- Test export authentication, method/format validation, ZIP datasets, formula-injection
  neutralisation, workbook sheets/charts, response headers, and browser bearer-token wiring.
- Run `make test` and `make build`.
