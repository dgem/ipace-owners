# Public Content and Positioning Prompt

Create or refine the public content pages for the I-PACE Owners' Advocacy Group.

## Goal

The public site should explain the group clearly, invite owners to participate, and establish a constructive evidence-led posture without overstating legal status or data certainty.

## Tone

- Serious, constructive, collaborative, and evidence-led.
- Use the display wordmark `i-Pace Owners` on the first line and `Advocacy Group` on the second line in the header brand area.
- Use `i-Pace Owners' Advocacy Group` for site-brand references where natural, while preserving `Jaguar I-PACE` when referring to the vehicle model.
- Speak as owners organising for fair and consistent outcomes. For launch, describe the public
  effort as UK-led while allowing non-UK owners to register interest separately.
- Avoid aggressive legal language on the homepage.
- Frame legal action only as a possible alternative that a constructive resolution may help avoid.
- Do not claim the group is formally incorporated, legally represented, or already operating a legal claim unless that is explicitly true.
- Do not imply submitted data is legally verified.
- Include "Speak softly and carry a big stick" on the About page or another appropriate explanatory page, not as the main homepage slogan.

## Public pages

The public presentation has two feature-flagged modes:

- `launch` is the deployed default and focuses on recruitment. Its main navigation contains
  Home, About, FAQ, and Join; authenticated identity/member controls remain available.
- `full` preserves Evidence, Methodology, Updates, technical homepage sections, and extended
  data-oriented calls to action. Reviewers can select it with `?site-mode=full`; the choice
  persists in session storage. `?site-mode=launch` restores launch mode.

This mode is a presentation/discoverability flag, not an authorization boundary. Protected
member data must always retain server-side Firebase ID-token verification. Full-mode content
must be hidden by default and only shown when the root document is explicitly in
`data-site-mode="full"`; keep a small critical head style and the main stylesheet aligned so
mobile browsers fail closed rather than showing both launch and full content.

Create or maintain:

- Home: clear proposition, why now, calls to join and submit vehicle data, and an integrated I-PACE-style vehicle image or equivalent original visual asset.
- About: purpose, approach, independence, constructive posture.
- FAQ: common owner questions, data use, membership, legal status, privacy, evidence process.
- Methodology: how evidence is collected, verified, anonymised, aggregated, and caveated.
- Updates index: list update posts from `src/updates/`.
- Individual update posts: Markdown with front matter.
- Contact: volunteer-run contact information and ways to help.
- Privacy: placeholder until formal review before broader vehicle/evidence data collection.
- Participation statement/terms: non-legal-action participation framing.

## Launch Homepage Detail

The default homepage should be a concise, mobile-first recruitment landing page that:

- Presents one organised owner voice and one primary Join call to action.
- Use the launch headline direction "I-PACE owners working together for fair outcomes" rather
  than duplicating separate scene-setting sections.
- For launch, the public ask is registration only. Ask owners for name, email, contact consent,
  and the non-legal-claim acknowledgement. Do not ask for vehicle details in launch mode, and
  do not imply that structured evidence collection is already the current public task.
- Include concise launch-visible "Why now?" context naming traction battery faults, recall or
  customer notice work, warranty uncertainty, and inconsistent support. Keep this high-level;
  do not restore recall-code/statistics-heavy content to launch mode.
- The launch "Why now?" section should use the same structured visual design language as the
  full homepage's issue list: responsive columns, `why-item` rows, and a clear callout. Its
  copy remains launch-specific and registration-only.
- Offers JLR a direct route to understand owner preferences and shape a broadly acceptable offer.
- Prefers a constructive agreement before Jaguar's next vehicle launch without inventing a date.
- Explains neutrally that data-protection restrictions prevent JLR sharing customer identities,
  so an independent consent-based group is needed to bring owners together.
- Clarifies that this is an advocacy group pursuing an outcome, not another discussion forum.
- States that the group is not an open comment board and is not intended to fragment into
  individual settlement demands.
- Explains that this is a UK-led initiative; overseas owners can register interest, but any
  market-specific representation or future legal route must be scoped separately.
- Keeps credible escalation available without confrontational language.
- May state that legal counsel exists within the group to inform the process and assess options,
  but must also state that this does not make the group a law firm, a current legal claim, or
  formal legal representation for members.
- States that joining does not enrol an owner in legal action; any future proposal is explained
  separately and each owner chooses whether to take part.
- Does not lead with recall codes, evidence methodology, statistics, or vehicle-data requests.
- Explains that vehicle details and fuller structured evidence may be requested in the coming
  weeks, separately and with clear purpose, if needed.

## Full Homepage Detail

The retained full-mode homepage should include:

- Headline around owners working together for fair outcomes.
- Context for traction battery faults, recall work, degradation, warranty uncertainty, and
  cars moving beyond the 8-year battery warranty.
- Feature cards for Collective Voice, Constructive Approach, and Evidence Based.
- A "Why now?" section mentioning traction battery faults, H447/H570/H571/H572 campaign
  and recall concerns, HV battery work and module replacements, battery State of Health concerns, vehicles
  approaching or beyond the 8-year / 100,000-mile battery warranty, and inconsistent owner
  experiences around support, loan cars, parts delays, and goodwill payments.
- Place "Why now?" immediately after the hero and arrange its issue items in two responsive
  columns. Follow it with the three feature cards, then a two-column section with the "Get
  started" panel first and the "This is not a legal action" callout second.

## About Detail

The About page should include:

- Purpose of the group.
- Immediate focus on I-PACE traction battery faults, recall/customer-notice work, warranty
  uncertainty, and inconsistent support.
- Constructive engagement with JLR.
- Independence from JLR.
- Principles: evidence based, respectful and constructive, transparent with members,
  independent of JLR, prepared to escalate if required.
- Goals: build the largest possible UK I-PACE owner community, collect robust owner data,
  understand the scale of battery and recall issues, engage JLR at senior level, seek fair
  and consistent treatment, record overseas interest separately where useful, avoid fragmented
  complaints and unnecessary legal action where possible, and protect owner confidence and
  vehicle value.

## FAQ Detail

Include at least:

- Is this a legal action?
- Do you have legal counsel or legal input?
- Is this anti-JLR?
- Will my VIN be public?
- Will my personal data be shared?
- Can non-UK owners register interest in a UK-led initiative?
- Can members still pursue their own complaint or claim?
- What recalls are relevant?
- What is SoH?
- Can I join without submitting vehicle data?
- Why collect loan car, delay, payment and goodwill information?
- How will submitted data be verified?
- Why do I need to sign in?

## Privacy and Terms Detail

Privacy should explain what data may be collected, why it is collected, how anonymised
statistics may be used, that personal data should not be published publicly, that Join
membership submissions and signed-in vehicle basics are stored with Go Cloud Functions and
Firestore, that Cloud Storage is reserved for uploaded evidence files, that full VINs are
not stored, that uploads are not implemented in the MVP, that Firebase Authentication is
used for authentication, and that a more formal privacy policy is needed before broader live
evidence data collection.

Terms / Participation Statement should say the group is informal unless a formal organisation
is created, the site does not provide legal advice, submitting data does not create a client
relationship with any lawyer, owners should submit accurate information to the best of their
knowledge, and data may be used in anonymised aggregate reporting if consent is given.

## Content constraints

- Do not include real owner data.
- Do not include unverifiable statistics as real figures.
- The launch homepage may show the live aggregate count of unique, non-excluded registered
  members since the 17 July 2026 launch as a compact, non-interactive racing winner's garland
  directly below the hero Join CTA on both wide and narrow screens. Keep the CTA and garland
  as one vertical unit above the fold, with the social proof visually subordinate to the CTA
  and without button affordance. Model it closely on a clean gold car-decal winner's wreath:
  a broad oval with
  sparse, large leaves, generous central space, and long lower branches that cross and extend
  beyond the join. Use flat gold without dark edging, adding subtle darker-gold veins to only a
  few leaves for definition. Do not use flags, a filled badge, wax-stamp treatment, heavy
  outlines, or a decorative tie. Display only the large, unoutlined gold count inside the
  wreath and optically centre it slightly above the geometric midpoint. Place compact white
  `Registered members` text and `Since 17 July 2026` as two quiet lines below and outside the
  garland rather than squeezing supporting copy into the artwork. Format the count with UK
  thousands separators and reduce only its font size at four, five, and six-plus digits so
  growth beyond 1,000 members cannot collide with the wreath.
- Mark placeholder figures and illustrative examples clearly.
- Do not use Jaguar/JLR logos, badges, readable plates, copyrighted press imagery, or other unlicensed manufacturer assets.
- If using a vehicle image, make it original/licensed/generated, avoid visible brand marks, and provide meaningful alt text.
- Keep public calls to action clear but not coercive.
- Include simple social share links for common channels such as X, Facebook, LinkedIn, and
  WhatsApp. Use accessible link text, open external share targets safely with
  `rel="noopener noreferrer"`, and keep the UI restrained.

## Validation

- Run `make build`.
- Check that every page has front matter with `layout`, `title`, and `description`.
- Check links among Join, Submit Vehicle Data, Evidence Dashboard, FAQ, Methodology, Privacy, and Terms.
