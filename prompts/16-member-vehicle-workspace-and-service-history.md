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

The Go Function must verify Firebase Auth, vehicle ownership, and existing-record ownership.
Store canonical records in Firestore `serviceEvents`, preserve creation/review metadata on
edits, and regenerate the private member snapshot after every successful write. Do not add
these records to public statistics until consent, moderation, and publication rules exist.

## Tests

- Test unauthenticated rejection and input validation.
- Test ownership predicates for both user and vehicle IDs.
- Test Firebase Hosting route and browser bearer-token wiring.
- Test tab semantics, full-width workspace markup, graph accessibility, and add/edit controls.
- Run `make test` and `make build`.
