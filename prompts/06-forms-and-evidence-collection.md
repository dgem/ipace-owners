# Forms and Evidence Collection Prompt

Implement or refine the membership and vehicle data collection form UX.

## Goal

Provide accessible multi-step forms that help owners submit structured evidence later, while clearly preventing and disclosing that no data is currently stored until backend persistence exists.

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
- Attach navigation behavior to every `[data-next]` and `[data-prev]` button in the form, not only the first matching button.
- Move focus to the current step heading after navigation.
- Remove hidden-step controls from the tab order.
- Respect `prefers-reduced-motion`.
- Prevent default submission until backend prompts explicitly add persistence.
- Show a clear placeholder result explaining that no data was sent or stored.
- Validate required text, email, select, checkbox, and radio controls according to their actual user state. Required checkboxes must be checked; required radio groups must have a checked option.
- When submission completes, hide all step navigation containers and all form steps before showing the placeholder result.

## Join form

Collect:

- Contact details.
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

## Data safety

- Do not store data in Git.
- Do not send data to a backend until a backend prompt implements it.
- Do not store raw VINs in public static files.
- Make evidence uploads a placeholder until server-side validation and storage exist.

## Validation

- Run `npm run build`.
- Keyboard-test next, previous, validation, and final placeholder result.
- Specifically test progressing past the second step of the Join form.
- Confirm required fields have accessible error messages.
