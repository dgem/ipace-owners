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
- On large screens, place form-level notices or storage-status callouts beside the form instead of above it when that improves first-step visibility.
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
  Identity email flow.
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
  - `data-database-auth-error` — shown if a signed-in submission lacks a valid Identity JWT
  - `data-database-submission-id` — filled with the stored submission reference when present
  - `data-registration-link-sent` — shown once the API call succeeds
  - `data-registration-error` — shown if the API call fails
- Join completion should not offer vehicle or account CTAs in the result panel. For guests,
  tell the user to open the email sign-in link first. For already signed-in users, confirm
  the Join answers were saved; onward navigation remains available through the normal
  signed-in header controls.
- Validate required text, email, select, checkbox, and radio controls according to their actual user state. Required checkboxes must be checked; required radio groups must have a checked option.
- On the Join form, the final "Join the Group" submit button should be enabled only after
  the contact consent and not-a-legal-claim acknowledgement checkboxes are ticked.
- When submission completes, hide all step navigation containers and all form steps before showing the placeholder result.

## Join form

Collect:

- Contact details.
- Default country / region to United Kingdom while keeping non-UK owner options available.
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

The member dashboard must show one selected car at a time, with tabs for switching cars and
a separate Add vehicle command. Show the selected car's SoH measurement history as an
accessible graph and table, with a button that reveals the form for appending another
reading. Post updates to `POST /api/submit-soh` with the vehicle ID, SoH percentage,
measurement date, optional mileage, and source. Never overwrite or discard earlier readings;
degradation analysis depends on the time series.

Below the SoH history, show an editable service/fault timeline. Members can add service,
fault, repair, recall, inspection, or other records with date, optional mileage, summary,
details, and status. Post additions and edits to `POST /api/upsert-service-event`; the server
must verify the signed-in member owns both the vehicle and any existing record being edited.

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
- H570/H571/H572 recall status and outcomes.
- Dealer/service experience.
- Loan car and mobility support.
- Goodwill, payments, refusals, or warranty responsibility.
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
