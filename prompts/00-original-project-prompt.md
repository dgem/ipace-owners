# Original Project Prompt

This file stores the initial prompt used with GitHub Copilot during the repository creation
phase. It is preserved as historical source material.

The maintained, split prompt set lives in `01-` through `20-` in this directory. Prefer those
files for future rebuilds and feature work; the Netlify-era body below remains unchanged as
historical source material.

---

## Initial Copilot prompt

```text
Create an initial MVP pull request for the repository `github.com/dgem/ipace-owners`.

Project: I-PACE Owners' Advocacy Group website.

Goal: build a static, mobile-first, accessible public website with Markdown-authored content, Netlify Identity login, gated placeholder member/admin pages, and multi-step data collection forms. This first PR should establish the site architecture, styling, public pages, content model, Netlify Identity integration, and frontend form UX. Do not implement real backend persistence yet.

Preferred stack:
- Eleventy / 11ty static site generator.
- Author public content pages in Markdown with front matter.
- Generate semantic, responsive, mobile-first HTML.
- Use custom CSS, not Tailwind.
- Include a modern CSS reset / compatibility baseline.
- Use CSS custom properties for colour, spacing, typography, shadows and radii.
- Use plain JavaScript only where useful:
  - mobile menu toggle
  - Netlify Identity UI state
  - multi-step form navigation
  - client-side form progress indicators
  - light validation helpers
- Target Netlify deployment.
- Add Netlify Identity in this PR.
- Prepare for future Netlify Functions and Netlify Blobs, but do not implement storage yet.

Important implementation choices:
- Do not use Tailwind.
- Do not use Bootstrap.
- Do not use React, Vue, Svelte or another heavy frontend framework.
- Do not implement a database layer in this PR.
- Do not persist submitted data in Git.
- Do not store form submissions locally except perhaps temporary in-memory/client-side state while navigating the form.
- Do not include real owner data.
- Do not use Jaguar/JLR logos or copyrighted imagery unless already licensed.
- Keep HTML semantic and accessible.

NPM / NPX expectations:
- Use npm scripts for repeatable project commands:
  - `npm run dev`
  - `npm run build`
  - `npm run clean`
- It is fine to mention `npx @11ty/eleventy` in the README as a one-off way to run the local Eleventy binary, but Netlify and contributors should use npm scripts.
- Include a `package-lock.json`.

Tone and positioning:
- The group should sound constructive, evidence-led, collaborative and serious.
- The site should make clear that the aim is to work positively with JLR where possible, while organising owners so there is credible data and a collective voice.
- Avoid aggressive legal language on the homepage.
- Legal action should be framed only as a possible alternative that a constructive resolution may help avoid.
- Include the phrase "Speak softly and carry a big stick" somewhere appropriate, probably on the About page, not as the main homepage slogan.
- Do not make claims that the group is already formally incorporated or legally represented.
- Do not imply that submitted data is legally verified.

Recommended colour scheme:
Use a calm, credible, technical-feeling palette. Define it in CSS custom properties.

Suggested palette:
- Background: #f7f8fb
- Surface: #ffffff
- Surface muted: #eef2f7
- Text: #111827
- Text muted: #4b5563
- Border: #d9e1ea
- Primary/navy: #12324a
- Primary dark: #0b2233
- Accent/teal: #0f766e
- Accent soft: #ccfbf1
- Warning/amber: #b45309
- Danger/red: #b91c1c
- Success/green: #15803d
- Focus outline: #2563eb

Design direction:
- Mobile-first.
- Clean, calm, modern, credible.
- Spacious but not flashy.
- Accessible colour contrast.
- Rounded cards with subtle borders.
- Use a strong navy header/footer and restrained teal accents.
- Avoid an angry campaign look.
- Avoid luxury car branding imitation.
- Prefer system fonts:
  `font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;`

CSS requirements:
Create `src/assets/css/site.css` or `src/assets/css/input.css` and copy it to the output build.

Include:
1. Modern reset:
   - `box-sizing: border-box`
   - remove default body margin
   - responsive media defaults
   - form controls inherit font
   - smooth-ish but accessible focus styling
   - respect `prefers-reduced-motion`
2. Design tokens using `:root`.
3. Base typography:
   - responsive line heights
   - sensible heading scale
   - readable measure for Markdown content
4. Layout utilities:
   - `.container`
   - `.section`
   - `.cluster`
   - `.stack`
   - `.grid`
   - `.two-column`
   - `.visually-hidden`
5. Components:
   - header
   - nav
   - mobile nav
   - hero
   - cards
   - stat cards
   - callouts
   - buttons
   - badges
   - forms
   - fieldsets
   - form steps
   - progress indicator
   - dashboard panels
   - footer
6. Print-friendly basics if easy.

Initial CSS should be production-quality enough to make all generated pages look coherent without Tailwind.

Site pages to create:
1. Home
2. About
3. Join
4. Submit Vehicle Data
5. Evidence Dashboard
6. Evidence Methodology
7. FAQ
8. Updates
9. Contact
10. Privacy
11. Terms / Participation Statement
12. Member Dashboard placeholder
13. Admin Review Queue placeholder
14. Account / Sign in page

Navigation:
Public nav:
- Home
- About
- Evidence
- Methodology
- FAQ
- Updates
- Join

Member nav:
- Dashboard
- My Vehicle
- My Evidence
- Account

Admin nav:
- Review Queue
- Submissions
- Evidence Files
- Public Stats
- Exports
- Members
- Settings

Member and admin pages:
- Add static placeholder pages.
- Gated in the frontend using Netlify Identity state.
- If the user is not logged in, show a sign-in/register prompt instead of the page content.
- For admin pages, check for an `admin` role in the Netlify Identity user app_metadata/roles where possible. If there is no admin role, show an access-restricted message.
- Make clear in comments/docs that frontend gating is not sufficient for future real data access. Future Netlify Functions must verify Identity JWTs server-side.

Netlify Identity requirements:
- Include Netlify Identity widget or `@netlify/identity` / `netlify-identity-widget`, whichever is simplest and reliable for an Eleventy static site.
- Add login / signup / logout controls in the header.
- Add an Account page where a logged-in user can see placeholder account information.
- Add JavaScript to:
  - initialise Netlify Identity
  - update header UI when logged in/out
  - show login modal from "Sign in" buttons
  - redirect or reveal gated content based on auth state
- Include a README section explaining that Netlify Identity must be enabled in the Netlify UI after deployment.
- Do not create custom password storage.

Homepage content requirements:
- Headline around owners working together for fair outcomes.
- Explain that the site is for Jaguar I-PACE owners affected by traction battery faults, recall work, battery degradation, warranty uncertainty and cars moving beyond the 8-year battery warranty.
- Include three feature cards:
  1. Collective Voice
  2. Constructive Approach
  3. Evidence Based
- Include a "Why now?" section mentioning:
  - traction battery faults
  - H570 / H571 / H572 recall concerns
  - HV battery work and module replacements
  - battery State of Health concerns
  - vehicles approaching or beyond the 8-year / 100,000-mile battery warranty
  - inconsistent owner experiences around support, loan cars, parts delays and goodwill payments
- Primary CTA: "Join the Group"
- Secondary CTA: "Submit Vehicle Data"

About page content requirements:
- Explain the purpose of the group.
- Emphasise constructive engagement with JLR.
- Emphasise independence from JLR.
- Include principles:
  - Evidence based
  - Respectful and constructive
  - Transparent with members
  - Independent of JLR
  - Prepared to escalate if required
- Include goals:
  - Build the largest possible community of I-PACE owners
  - Collect robust owner data
  - Understand the scale of battery and recall issues
  - Engage JLR at senior level
  - Seek fair and consistent treatment
  - Avoid fragmented complaints and unnecessary legal action where possible
  - Protect owner confidence and vehicle value

Join page:
Create a multi-step form. It does not need to persist data yet.
Use semantic HTML with fieldsets, legends, labels, hints and accessible error containers.

Steps:
1. Contact details
2. Ownership status
3. Skills / volunteering
4. Consent and review

Fields:
- Name
- Email
- Country / region
- Relationship to vehicle:
  - Current owner
  - Former owner
  - Prospective buyer
  - Trade / specialist
  - Other
- Vehicle ownership:
  - I own one I-PACE
  - I own more than one
  - I no longer own one
  - I am helping someone who owns one
- Would you like to help run the group? Multi-select:
  - Legal
  - Technical / battery expertise
  - Data analysis
  - Media / PR
  - Web / admin
  - Consumer rights
  - Dealer contacts
  - General volunteering
- Consent checkboxes:
  - I agree to be contacted about the owners' group.
  - I understand this is not currently a legal claim.
  - I consent to my submitted data being used in anonymised aggregate analysis.

Submit Vehicle Data page:
Create a multi-step form. It does not need to persist data yet.
Use progressive enhancement-friendly HTML. Group fields with fieldsets and legends.
Include a progress indicator, Back/Next buttons, Save later placeholder button, and final Review step.
If JavaScript is disabled, the form should still show all sections in a usable single-page layout.

Steps:

1. Vehicle details
- VIN
- Model year
- Registration, optional
- Country
- Mileage
- Ownership since
- First registration date, if known

2. Battery health
- Current State of Health, if known
- Date SoH was measured
- Mileage at SoH measurement
- Source of SoH:
  - Dealer report
  - Diagnostic app / OBD
  - Service paperwork
  - JLR communication
  - Estimate / unsure

3. HV battery work
Assume work may have been done, but capture the details neutrally.
- Has the car had HV battery work?
  - No
  - Yes
  - Unsure
- Type of work:
  - Software update only
  - One or more modules replaced
  - Full HV battery replaced
  - Battery inspection only
  - Other / unknown
- Number of modules replaced, if known
- Date fault first reported
- Date car went to dealer, if known
- Date car returned, if known
- Approximate total days off road
- Approximate days waiting before work started
- Approximate days while work was being carried out
- Was this the first HV battery issue?
- Has the fault reoccurred?

4. Additional warranty cover at time of fault
Do not ask owners to manually answer whether the car was inside the original manufacturer warranty or inside the 8-year / 100,000-mile battery warranty, because those can be inferred later from vehicle age and mileage.
Instead ask:
- At the time the battery issue was reported, did the vehicle have any additional warranty cover?
  - Extended manufacturer warranty
  - Third-party warranty
  - No additional warranty cover
  - Unsure
- Warranty provider, if known
- Warranty policy name, if known
- Expiry date, if known
- Was the claim accepted under this warranty?
  - Yes, fully
  - Yes, partially
  - No
  - Initially refused, later accepted
  - Still pending
  - Not applicable
  - Unsure

5. Recall status
For H570, H571 and H572 capture:
- Status:
  - Completed
  - Booked
  - Awaiting parts / appointment
  - Dealer says not applicable
  - Owner declined
  - Unknown
- Date owner notified, optional
- Date booked, optional
- Date completed, optional
- Dealer / service centre, optional
- Mileage at completion, optional
- Did the recall lead to any battery warning, restriction or fault?
  - No
  - Yes
  - Unsure

6. Work experience and support
- Did the work go as planned?
  - Yes, completed as expected
  - Mostly, but with minor issues
  - No, there were significant issues
  - Work is still ongoing
  - Unsure
- Difficulties encountered, multi-select:
  - Long wait for appointment
  - Long wait for parts
  - Long wait for diagnosis
  - Vehicle off road longer than expected
  - Work did not fix the problem
  - Fault returned after work
  - Additional work was needed
  - Poor communication from dealer
  - Poor communication from JLR
  - Disagreement over warranty / responsibility
  - Asked to make a payment
  - Loan car issue
  - Other
- Anything else we should know? Free text.

7. Loan / courtesy car
- Was a loan or courtesy car provided?
  - Yes, for the whole period
  - Yes, but only for part of the period
  - Yes, but only after I asked or escalated
  - No
  - Not needed
  - Unsure
- Was the loan car broadly suitable?
  - Yes
  - No
  - Unsure
- Loan car type:
  - I-PACE or comparable EV
  - Other EV
  - Petrol / diesel / hybrid
  - Hire car
  - Dealer courtesy car
  - Other
  - Unsure
- Were you charged anything for the loan car?
  - No
  - Yes
  - Deposit only
  - Insurance/admin fee only
  - Unsure
- Approximate amount charged, if any

8. Payments, goodwill and expenses
- Did you have to pay anything towards diagnosis, repair, recovery, admin or transport?
  - No
  - Yes
  - Asked to pay, but later waived/refunded
  - Still disputed
  - Unsure
- Approximate amount
- What was the payment for?
- Were you offered any goodwill payment, reimbursement or contribution?
  - No
  - Yes
  - Requested but refused
  - Still pending
  - Unsure
- Goodwill/reimbursement types:
  - Fuel costs
  - Charging costs
  - Mileage costs
  - Hire car costs
  - Taxi / public transport
  - Recovery costs
  - Insurance/admin cost
  - Repair contribution
  - Cash goodwill payment
  - Warranty extension
  - Other
- Approximate amount offered or paid

9. Responsibility acceptance
- Were there any issues with JLR, the dealer or warranty provider accepting responsibility?
  - No
  - Yes, initially refused
  - Yes, partially accepted only
  - Yes, still disputed
  - Unsure
- Who was involved?
  - JLR customer services
  - Dealer / retailer
  - Extended warranty provider
  - Third-party warranty provider
  - Finance company
  - Insurance company
  - Other
- How was it resolved?
  - Accepted without escalation
  - Accepted after escalation
  - Partially accepted
  - Refused
  - Still unresolved
  - Not applicable

10. Evidence upload placeholder
Do not implement file upload persistence yet. Create a visually clear placeholder explaining that a future version will allow uploads of:
- Dealer reports
- Battery health reports
- Recall paperwork
- Invoices
- Warranty decisions
- JLR correspondence
- Recovery/hire car/expense evidence
Include copy telling owners to redact personal information they do not want stored.

11. Consent
- Use my data anonymously in public statistics.
- Allow organisers to contact me for clarification.
- Allow my anonymised data to be included in reports to JLR.
- I may be willing to provide evidence for media/legal/consumer rights purposes later.

12. Review
Show a static/dynamic review summary of entered information if feasible. If that is too much for the first PR, include a placeholder review panel and make the structure ready for it.

Multi-step form JS requirements:
- Generic script that works from data attributes, not hardcoded to one form only.
- Add `data-multistep-form`, `data-step`, `data-next`, `data-prev`, `data-progress`.
- Update visible step and progress text.
- Keep all fields in the DOM.
- Do not submit anywhere yet.
- On final submit, prevent default and show a clear "Submission storage is not enabled yet" message.
- Ensure keyboard users can navigate.
- Move focus to the next step heading after step change.
- Respect reduced motion.

Evidence Dashboard page:
Create a public anonymised dashboard layout using placeholder/sample data only.
Include stat cards:
- Owners contributed
- Vehicles with HV battery work
- Average reported SoH
- Recall completion rate
- Vehicles outside battery warranty
- Vehicles with repeat battery issues
- Average days off road
- Loan car provided
- Goodwill/payment issues reported
Include simple responsive placeholder chart panels using accessible HTML/CSS, not canvas libraries:
- State of Health distribution
- Model year distribution
- HV battery work by model year
- Modules replaced
- Recall completion
- Inside vs outside 8-year / 100,000-mile battery warranty
- Average time off road
- Loan car provision
- Payment/goodwill outcomes
Include clear disclaimer:
"Data is submitted voluntarily by owners. Entries will be marked by verification level: self-reported, document supplied, reviewed, duplicate-checked, or excluded from public statistics. Public figures are anonymised and aggregated."

Evidence Methodology page:
Explain:
- How data is collected
- How data will be verified
- Verification levels:
  - Self-reported
  - Document supplied
  - Reviewed by organiser
  - Duplicate checked
  - Excluded from public statistics
- What is published
- What is not published
- Known limitations
- Why structured data is more credible than forum anecdotes
Include privacy note:
Do not publish full VINs, email addresses, owner names, registrations or uploaded documents without explicit permission.

FAQ page:
Include at least these questions:
- Is this a legal action?
- Is this anti-JLR?
- Will my VIN be public?
- Will my personal data be shared?
- Can non-UK owners join?
- What recalls are relevant?
- What is SoH?
- Can I join without submitting vehicle data?
- Why collect loan car, delay, payment and goodwill information?
- How will submitted data be verified?
- Why do I need to sign in?

Updates page:
Create a simple list of placeholder updates:
- Owner group launched
- First owner data collection planned
- Methodology published
- Evidence dashboard prototype published

Privacy page:
Create a plain-English placeholder privacy policy. It should explain:
- What data may be collected
- Why it is collected
- How anonymised statistics may be used
- That personal data should not be published publicly
- That uploads are not implemented in the first PR
- That a more formal privacy policy will be needed before live data collection
- That Netlify Identity is used for authentication

Terms / Participation Statement:
Create a placeholder participation statement saying:
- The group is an informal owners' advocacy initiative unless and until a formal organisation is created.
- The site does not provide legal advice.
- Submitting data does not create a client relationship with any lawyer.
- Owners should submit accurate information to the best of their knowledge.
- Data may be used in anonymised aggregate reporting if consent is given.

Suggested project structure:
- `src/pages/*.md` for Markdown-authored static pages.
- `src/pages/join.njk` for the multi-step join form if Markdown is too limiting.
- `src/pages/submit-vehicle-data.njk` for the multi-step vehicle form.
- `src/updates/*.md` for update posts.
- `src/_data/site.json`
- `src/_data/navigation.json`
- `src/_includes/layouts/base.njk`
- `src/_includes/layouts/page.njk`
- `src/_includes/layouts/form-page.njk`
- `src/_includes/partials/header.njk`
- `src/_includes/partials/footer.njk`
- `src/_includes/partials/nav.njk`
- `src/_includes/partials/card.njk`
- `src/_includes/partials/callout.njk`
- `src/assets/css/site.css`
- `src/assets/js/main.js`
- `src/assets/js/identity.js`
- `src/assets/js/multistep-form.js`
- `public/images/`
- `netlify/functions/.gitkeep` for future functions
- `netlify.toml`
- `.eleventy.js`
- `package.json`
- `package-lock.json`
- `README.md`
- `docs/architecture.md`

Build requirements:
- `npm install`
- `npm run dev` should start the local Eleventy development server.
- `npm run build` should generate the production site.
- `npm run clean` should remove `_site`.
- Netlify publish directory should be `_site`.

Netlify configuration:
Create `netlify.toml`:
- build command: `npm run build`
- publish directory: `_site`
- functions directory: `netlify/functions`
- include any useful security headers if appropriate, but do not break Netlify Identity.

Future architecture note:
Create `docs/architecture.md` explaining the intended future architecture:
- Netlify Identity for owner/admin authentication.
- Netlify Functions for validation, permissions and form handling.
- Netlify Blobs for JSON records and uploaded evidence.
- Potential future Netlify Database/Postgres if querying becomes important.
- Raw VINs and personal data should not be stored in public static files.
- Use an HMAC with a secret pepper for VIN deduplication in future backend code, not a plain SHA-256 hash.
- Public pages should only show anonymised aggregate data.
- Frontend gating is not enough for real private data; future Functions must verify Identity JWTs server-side.

README requirements:
Include:
- Project overview.
- Local setup.
- `npm install`
- `npm run dev`
- `npm run build`
- Optional `npx @11ty/eleventy` note for running the local Eleventy binary.
- Netlify deployment steps.
- How to enable Netlify Identity in the Netlify UI.
- Current limitations:
  - form submission not yet persisted
  - evidence uploads not yet implemented
  - admin review queue is placeholder
  - privacy policy is placeholder pending live data collection

Acceptance criteria:
- The site builds successfully.
- The homepage, about page, join page, vehicle data form, evidence dashboard, methodology page, FAQ, updates, privacy, terms, account, member dashboard and admin review queue pages exist.
- Markdown pages render through shared layouts.
- The design is responsive and mobile-first.
- Custom CSS provides a complete coherent design without Tailwind.
- Navigation works on mobile and desktop.
- Netlify Identity login/signup/logout UI is present.
- Logged-in/member placeholders respond to Identity state.
- Admin placeholder checks for an admin role where possible.
- Multi-step forms work with JavaScript and remain usable without JavaScript.
- Forms are accessible and grouped with labels/fieldsets.
- There is a clear path for future Netlify Functions and Blobs integration.
- README explains how to run, build and deploy the project.
```
