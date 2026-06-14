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
- Without JavaScript, all form steps should remain visible and usable as a single-page form.
- With JavaScript, show one step at a time.
- On large screens, place form-level notices or storage-status callouts beside the form instead of above it when that improves first-step visibility.
- Keep the first actionable form fields visible as high as practical; avoid stacking large notices above progress indicators.
- Keep step navigation, especially the first Next button, visible without excessive scrolling on common laptop-height viewports.
- Use compact spacing or two-column field grouping on desktop when it improves form progression without harming readability.
- Attach navigation behavior to every `[data-next]` and `[data-prev]` button in the form, not only the first matching button.
- Move focus to the current step heading after navigation.
- Remove hidden-step controls from the tab order.
- Respect `prefers-reduced-motion`.
- Prevent default submission until a specific backend integration exists for that form.
- Show a clear placeholder or handoff result explaining what happened. For forms without a
  backend, explain that no data was sent or stored. For the Join form, explain that the
  membership answers are saved with Netlify Forms and the name/email are used for the
  Identity email flow.
- Configure the Join form as a Netlify Form:
  - include `name="join"`, `method="POST"`, `action="/join/"`, `data-netlify="true"`,
    `netlify-honeypot`, and hidden `form-name=join`;
  - preserve no-JavaScript submission as a normal HTML form;
  - in the JavaScript-enhanced flow, submit `FormData` as
    `application/x-www-form-urlencoded` so Netlify Forms receives the same fields without
    leaving the multi-step result screen.
- After successful Join form validation, `identity.js` calls
  `POST /.netlify/functions/send-magic-link` with the email and name collected in
  step 1. The function sends a confirmation or recovery email via Netlify Identity.
  The email address **is** sent to Netlify Identity; detailed join answers (ownership,
  skills, consent) are saved with Netlify Forms. The result screen independently reports
  whether the form save and magic-link email handoff succeeded.
- Do not claim "no data was sent or stored" in the Join result. Clarify that the Join
  answers are saved with Netlify Forms and the email address is sent to Netlify Identity.
- The result area uses these data attributes for state management:
  - `data-registration-guest` — wrapper shown for unauthenticated users
  - `data-registration-signed-in` — shown if user is already logged in
  - `data-registration-email` — filled with the collected email address
  - `data-registration-form-saved` — shown once Netlify Forms accepts the Join submission
  - `data-registration-form-error` — shown if Netlify Forms submission fails
  - `data-registration-link-sent` — shown once the API call succeeds
  - `data-registration-error` — shown if the API call fails
- Validate required text, email, select, checkbox, and radio controls according to their actual user state. Required checkboxes must be checked; required radio groups must have a checked option.
- When submission completes, hide all step navigation containers and all form steps before showing the placeholder result.

## Join form

Collect:

- Contact details.
- Default country / region to United Kingdom while keeping non-UK owner options available.
- Owner/relationship status.
- Optional skills and volunteering.
- Consent to contact.
- Acknowledgement that joining is not a legal claim.
- Optional consent for anonymised aggregate analysis.

## Vehicle data form

Collect structured, optional evidence fields such as:

- VIN, registration, country, model year, ownership dates, mileage.
- Battery State of Health and source.
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

Include a Save later placeholder button or copy where practical, but do not implement
storage until backend persistence exists.

The review step may be a placeholder in the MVP, but the structure should be ready for a
future summary of entered information.

## Data safety

- Do not store data in Git.
- Do not send form data to a backend until a backend prompt implements it for that specific
  flow. The current exceptions are:
  - the Join form's Netlify Forms submission, which stores membership interest and consent;
  - the Join form's `send-magic-link` call, which sends email and name to Netlify Identity
    via a same-origin Function.
- Do not store raw VINs in public static files.
- Make evidence uploads a placeholder until server-side validation and storage exist.

## Validation

- Run `npm run build`.
- Keyboard-test next, previous, validation, and final placeholder result.
- Specifically test progressing past the second step of the Join form.
- Confirm required fields have accessible error messages.
