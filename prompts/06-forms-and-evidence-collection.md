# Forms and Evidence Collection Prompt

Implement or refine the membership and vehicle data collection form UX.

## Goal

Provide accessible multi-step forms that collect membership interest now, help owners submit structured evidence later, and clearly disclose what is and is not sent or stored at each stage.

## JavaScript behavior

- Use `src/assets/js/multistep-form.js`.
- Keep the controller generic and data-attribute driven.
- Use these attributes:
  - `data-multistep-form`
  - `data-step`
  - `data-step-heading`
  - `data-prev`
  - `data-next`
  - `data-submit`
  - `data-progress`
  - `data-progress-fill`
  - `data-progress-text`
  - `data-progress-step`
  - `data-submit-result`
  - `data-enable-when-checked` for submit buttons that must remain disabled until named
    consent checkboxes are ticked
- Without JavaScript, all form steps should remain visible and usable as a single-page form.
- With JavaScript, show one step at a time.
- Place form-level notices or storage-status callouts below the form, not beside or above it.
- Keep the first actionable form fields visible as high as practical; avoid stacking large notices above progress indicators.
- Keep step navigation, especially the first Next button, visible without excessive scrolling on common laptop-height viewports.
- Do not scroll the viewport on every step change by default. Only enable
  `data-scroll-on-step-change` as an exception for unusually long forms where the current
  step would otherwise be lost.
- Use compact spacing or two-column field grouping on desktop when it improves form progression without harming readability.
- Attach navigation behavior to every `[data-next]` and `[data-prev]` button in the form, not only the first matching button.
- Move focus to the current step heading after navigation.
- Remove hidden-step controls from the tab order.
- Respect `prefers-reduced-motion`.
- Prevent default submission until a specific backend integration exists for that form.
- Show a clear placeholder or handoff result explaining what happened. For forms without a
  backend, explain that no data was sent or stored. For the Join form, explain that the
  membership answers are saved with `submit-join` and the name/email are used for the
  Firebase passwordless email flow.
- Configure backend-backed forms with `data-database-submit` pointing to the relevant
  `/api/*` Go Cloud Function. Use JSON `fetch` from `identity.js`, and send a Firebase ID
  token in the `Authorization` header when `data-database-requires-auth` is present.
- Configure the Join form to call only `POST /api/submit-join` from the browser. Do not
  also call `POST /api/send-magic-link` on Join completion.
- After successful Join form validation, `identity.js` calls `submit-join` with the form
  payload. The Function stores the Join answers and sends a confirmation or passwordless
  magic-link email via shared server-side Firebase code.
  The email address **is** sent to Firebase Authentication; detailed join answers (combined
  relationship/ownership status, skills, consent) are saved to Firestore. The result screen reports storage success
  and the `magicLinkSent` state returned by `submit-join`.
- Do not claim "no data was sent or stored" in the Join result. Clarify that the Join
  answers are saved to the structured data store and the email address is sent to Firebase Authentication.
- The result area uses these data attributes for state management:
  - `data-registration-guest` — wrapper shown for unauthenticated users
  - `data-registration-signed-in` — shown if user is already logged in
  - `data-registration-email` — filled with the collected email address
  - `data-database-success` — shown once the Function stores the submission
  - `data-database-error` — shown if the Function cannot store the submission
  - `data-database-auth-error` — shown if a signed-in submission lacks a valid Firebase ID token
  - `data-database-submission-id` — filled with the stored submission reference when present
  - `data-registration-link-sent` — shown once the API call succeeds
  - `data-registration-error` — shown if the API call fails
- Join completion should not offer vehicle or account CTAs in the result panel. For guests,
  tell the user to open the email sign-in link first. For already signed-in users, confirm
  the Join answers were saved; onward navigation remains available through the normal
  signed-in header controls.
- Validate required text, email, select, checkbox, and radio controls according to their actual user state. Required checkboxes must be checked; required radio groups must have a checked option.
- On the Join form, the final "Join the Group" submit button should be enabled only after
  the contact consent and not-a-legal-claim acknowledgement checkboxes are ticked. Render it
  with a `disabled` attribute in the initial HTML and keep a submit-time guard in JavaScript
  so the consent requirement holds even if initial enhancement is delayed.
- Implement conditional checkbox enablement by reading checkbox controls directly by
  `name` from the form DOM and checking their actual `checked` state; do not infer this
  from broad form-control collections or serialized form data.
- The Join page below-form callout should explain that Join answers are saved and that, after
  confirming the emailed sign-in link, members can sign in and add vehicle details, State of
  Health readings, and service/fault history for one or more I-PACEs. Do not describe
  vehicle evidence storage as inactive now that vehicle basics, SoH, and service/fault
  history are implemented.
- Informational callouts on form pages should sit below the form, not in a right-hand rail
  or above the first step. The form is the primary task; people who need context can continue
  reading below it.
- When submission completes, hide all step navigation containers and all form steps before showing the placeholder result.

## Join form

The Join page has two presentation variants that post to the same `POST /api/submit-join`
contract:

- Default `launch` mode is a single-step minimum-data form containing name, email, contact
  consent, the not-a-legal-claim acknowledgement, optional anonymised aggregate-publication
  consent, and Privacy/Participation links. Country, relationship, and skills are omitted
  and therefore retain their existing optional/empty backend defaults.
- During launch, the public registration ask is intentionally limited to name, email, and
  consent preferences. Do not describe vehicle data submission as the current public task; say
  that vehicle details may be requested separately in the coming weeks if needed.
- `full` mode retains the complete four-step form described below. It is available through the
  session-persisted `?site-mode=full` presentation flag.

Both variants must retain the honeypot, accessible validation, disabled-until-consented submit
button, one browser POST, automatic server-side magic-link handoff, and shared completion state.
Completion copy must not imply that vehicle data is required or expected immediately; any later
request for vehicle details should be explained separately.
Render completion as a compact, centred confirmation card rather than stretching it across the
form workspace. Keep emphasis inline within sentences and hide the pre-submit informational
callout once the result is shown so it does not repeat the next-step guidance.

Collect:

- Contact details.
- Default country / region to United Kingdom while keeping non-UK owner options available.
  Explain that launch is UK-led and that non-UK interest may be kept separate for regional
  scope or future handoff.
- One combined owner/relationship status field. Do not ask separate relationship and
  ownership questions that allow contradictory answers.
  Use options equivalent to:
  - current owner of one I-PACE;
  - current owner of more than one I-PACE;
  - former owner;
  - prospective buyer;
  - helping an owner;
  - trade / specialist;
  - other.
- Optional skills and volunteering.
- Consent to contact.
- Acknowledgement that joining is not a legal claim.
- Optional consent for anonymised aggregate analysis.

## Vehicle data form

The currently implemented vehicle form is the first database-backed slice. It requires a
signed-in Firebase Auth user and calls `POST /api/submit-vehicle-basics`.
It collects:

- VIN, registration, country, model year, ownership dates, mileage.
- Battery State of Health and source.

The UX must support members registering multiple vehicles. Treat this as an "add/register a
vehicle" flow and provide an obvious route to add another vehicle after saving.
The vehicle identifier requirement must be enforced client-side before leaving the vehicle
details step, and server-side in `SubmitVehicleBasics`. If the API rejects a save, show the
server-provided error in the result panel and do not show success-only actions such as "Add
another vehicle".
Use client-side validation to improve data quality without blocking legitimate edge cases:
hard-block future dates and VIN-only invalid VINs; show soft warnings for GB registration
formats and valid VINs that do not look like typical Jaguar/JLR VINs. Server-side validation
must remain authoritative for future dates and identifier requirements.

The vehicle form callout should explain that this page starts the vehicle record and that,
after saving, members can use My Data to add further SoH readings and service/fault history.
Do not use stale copy saying service/fault history is unavailable; only fuller recall,
repair, loan car, payment, goodwill and evidence upload collection should be framed as
future expansion until those fields exist.

The member dashboard must show one selected car at a time, with tabs for switching cars and
a separate Add vehicle command. Show the selected car's SoH measurement history as an
accessible graph and table, with a button that reveals the form for appending another
reading. Post updates to `POST /api/submit-soh` with the vehicle ID, SoH percentage,
measurement date, optional mileage, and source. Never overwrite or discard earlier readings;
degradation analysis depends on the time series.

Below the SoH history, show an editable service/fault timeline. Members can add service,
fault, repair, recall, inspection, or other records with date, optional mileage, summary,
details, and status. New records should default to `fault`, because fault reporting is the
main evidence workflow. Service/fault records should optionally capture related campaigns
(`H441`, `H448`, `H570`, `H571`, `H572`, other/unsure/none), final fix date, days from fault to
final fix, whether a courtesy vehicle was offered/provided, whether there was delay due to
parts, warranty cover in place, and responsibility/warranty dispute status. Post additions
and edits to `POST /api/upsert-service-event`; the server must verify the signed-in member
owns both the vehicle and any existing record being edited.

Full VINs must not be stored. The Function should create an HMAC using `VIN_PEPPER` and
store only the HMAC plus final six characters for reference.
The first vehicle-basics slice should require at least one vehicle identifier: VIN or
registration.
Place field-specific help text below the relevant input where practical, especially for
VIN and registration. Keep the copy plain-English and reassuring: explain where owners can
find the VIN, that VIN is optional when registration is provided, and that full VINs are
never stored.

Future slices should collect structured, optional evidence fields such as:

- High-voltage battery work, modules replaced, dates, days off road.
- Additional warranty cover.
- Broader H441/H448/H570/H571/H572 campaign status and outcomes beyond the per-event timeline.
- Dealer/service experience beyond the per-event timeline.
- Payment, goodwill, and expense records.
- Repeat faults.
- Notes and evidence upload placeholder.
- Consent and review.

The vehicle form should remain neutral and detailed. It should not ask owners to manually
answer whether the car was inside the original manufacturer warranty or inside the 8-year /
100,000-mile battery warranty, because those can be inferred later from age and mileage.

Include a Save later placeholder button or copy where practical for future longer evidence
flows.

The review step may be a placeholder in the MVP, but the structure should be ready for a
future summary of entered information.

## Data safety

- Do not store data in Git.
- Do not send form data to a backend until a backend prompt implements it for that specific
  flow. The current exceptions are:
  - the Join form's single `submit-join` call, which stores membership interest and consent
    in Firestore and sends email/name to Firebase Authentication via shared server-side
    magic-link code;
  - the signed-in vehicle basics form's `submit-vehicle-basics` call, which stores the
    initial vehicle and battery health slice in Firestore;
  - the signed-in member SoH form's `submit-soh` call, which appends a measurement after
    server-side vehicle ownership verification;
  - the member service/fault form's `upsert-service-event` call, which adds or edits a record
    after server-side ownership verification.
- Do not store raw VINs in public static files.
- Do not store full VINs in Firestore, Cloud Storage, or static JSON. Store an HMAC generated with `VIN_PEPPER` and only the
  final six characters for reference. If `VIN_PEPPER` is not configured, ignore the VIN
  when registration is present, and reject VIN-only submissions with a clear configuration
  message.
- Make evidence uploads a placeholder until server-side validation and storage exist.

## Validation

- Run `make build`.
- Run `make test` when changing form submission wiring, Function payloads, or validation.
- Keyboard-test next, previous, validation, and final placeholder result.
- Specifically test progressing past the second step of the Join form.
- Confirm required fields have accessible error messages.
