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

## Member surveys

Provide a protected member survey page at `/member/surveys/` and an admin CRUD workspace at
`/admin/surveys/`. Administrators can create, edit, list, and delete a survey with a title,
public Description and Call to action fields, an optional question/prompt, two to twelve options,
a single- or multiple-choice setting, inclusive whole-day start/end dates, and an aggregate-results
visibility setting. Each survey also has an explicit `draft` or `published` status; drafts are
visible only in the admin workspace and published surveys are eligible for member views. New
surveys default to draft and today through six days later (seven inclusive whole
days), while retaining editable dates. Description, CTA, and each option's longer description
accept a safe Markdown subset (bold, emphasis, links, paragraphs, and bullet lists); raw HTML is
displayed as text. Each option must also have a distinct, single-line plain-text name (maximum 120
characters) used in aggregate results, alongside its Markdown description (maximum 2,000
characters). Preserve and derive a sensible name from legacy label-only options when reading older
documents. On the response and preview cards show the name first; descriptions that span multiple
lines or overflow their visible space should be collapsed to two visible lines with an accessible
`...more` / `Show less` control.
Any number of options may offer an optional 250-character free-text explanation (for example,
two distinct `Other` options); store text only when the member supplies it against the relevant
selected option.
For multiple-choice surveys only, let administrators explicitly mark each eligible option with
an `Allow members to mark this as their preferred option` setting. A member may then optionally
mark one selected eligible option as their preferred outcome. Use a clearly labelled checkbox only
on selected eligible choices, keep at most one checked in the browser, and validate server-side
that it is selected, eligible, and that the survey is multiple-choice. Store its stable option ID
and include separate aggregate preferred counts.
Link the implemented survey workspace from the protected Admin dashboard; do not leave it as an
undiscoverable direct URL.

Place the existing-surveys history above the admin editor. Load it after server-side admin
verification, refresh it automatically every 30 seconds and when the tab regains focus, as well
as after every save or delete; do not require an administrator to press Refresh after signing in.
Every survey, including a draft, must have an admin-only `Preview / test` action. The preview uses
the member response layout and validates a test selection, text explanation, and optional preferred
choice without persisting a response or affecting counts/results. Drafts must stay absent from all
member endpoints; reject an attempted member response to a draft rather than relying on its hidden
listing.
On both member landing pages—`/member/dashboard/` and `/member/account/` (the destination of the
signed-in `My Data` header action)—place a prominent, plain-language callout before the main
workspace: when one or more surveys are open, name them and provide a primary `Take the survey`
action plus a `Past surveys` action only when closed surveys exist. Refresh this summary after
member verification, every minute, and on focus. The member survey page is a date-ordered
directory of every published survey, with All, Open, Upcoming, and Closed filters. Each item
shows whether the member has submitted, and provides Submit/Edit only while open, and View
results when permitted. The separate response page must use large, numbered, card-like
checkboxes/radio choices with a visibly selected state; it must not read as a wall of unstructured
text.
Keep results off the response form. After a successful first submission or amendment, redirect to
`/member/survey-results/?id={surveyId}` with an explicit saved confirmation and the aggregate
count-only results. The server must withhold aggregate counts for an open survey until that member
has submitted a response, preventing popularity-led voting; once a published survey is closed,
results may be visible to every member if the administrator enabled them. Provide an `Edit your
response` link back to the response page. `Past surveys` filters `/member/surveys/?filter=closed`;
it is not a separate route and must never point to a currently open survey.

The admin dashboard places its actionable tool grid before campaign and member-statistics panels,
which are reference information rather than the primary starting point for administrator work.

Use Firestore `surveys/{surveyId}` documents and a `responses/{uid}` subcollection so a signed-in
member has one replaceable response per survey. The member APIs must verify Firebase ID tokens
server-side, accept responses only while the survey is live, validate option IDs and the selected
cardinality, accept an optional explanation for every selected text-enabled option, and never expose
free-text responses in aggregate results. Return option counts and the signed-in member's own
answer; display counts only when the administrator enabled results. In particular, never expose
one member's written response to another member: free-text is retained for administrators to
review manually, not published with results. Give administrators a separate
`/admin/survey-results/?id={surveyId}` analysis page, linked from every survey in the admin
history. It may show aggregate counts and each submitted answer, including free text, but must
identify a respondent only by a consistently masked email address. Its CSV export must contain
only that masked respondent, UTC submission time, selected option IDs, the optional preferred
option ID and text responses in `option-id: text` form—never
a full email address, name, Firebase UID, or member/vehicle data. Neutralise spreadsheet formula
characters in the CSV's user-controlled cells before exporting them.
Register `/api/admin/surveys`, `/api/admin/survey-preview`, `/api/admin/survey-results`, `/api/member/surveys`, and `/api/member/survey-response` through
the shared `Api` function. Test survey definition validation, response validation, and inclusive
date boundaries, including the PII-safe CSV fields. Delete a survey's response subcollection before deleting its parent document.
