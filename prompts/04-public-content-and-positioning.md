# Public Content and Positioning Prompt

Create or refine the public content pages for the I-PACE Owners' Advocacy Group.

## Goal

The public site should explain the group clearly, invite owners to participate, and establish a constructive evidence-led posture without overstating legal status or data certainty.

## Tone

- Serious, constructive, collaborative, and evidence-led.
- Use the display wordmark `i-Pace Owners` on the first line and `Advocacy Group` on the second line in the header brand area.
- Use `i-Pace Owners Advocacy Group` for site-brand references where natural, while preserving `Jaguar I-PACE` when referring to the vehicle model.
- Speak as owners organising for fair and consistent outcomes.
- Avoid aggressive legal language on the homepage.
- Frame legal action only as a possible alternative that a constructive resolution may help avoid.
- Do not claim the group is formally incorporated, legally represented, or already operating a legal claim unless that is explicitly true.
- Do not imply submitted data is legally verified.
- Include "Speak softly and carry a big stick" on the About page or another appropriate explanatory page, not as the main homepage slogan.

## Public pages

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

## Homepage Detail

The homepage should include:

- Headline around owners working together for fair outcomes.
- Context for traction battery faults, recall work, degradation, warranty uncertainty, and
  cars moving beyond the 8-year battery warranty.
- Feature cards for Collective Voice, Constructive Approach, and Evidence Based.
- A "Why now?" section mentioning traction battery faults, H570/H571/H572 recall concerns,
  HV battery work and module replacements, battery State of Health concerns, vehicles
  approaching or beyond the 8-year / 100,000-mile battery warranty, and inconsistent owner
  experiences around support, loan cars, parts delays, and goodwill payments.

## About Detail

The About page should include:

- Purpose of the group.
- Constructive engagement with JLR.
- Independence from JLR.
- Principles: evidence based, respectful and constructive, transparent with members,
  independent of JLR, prepared to escalate if required.
- Goals: build the largest possible I-PACE owner community, collect robust owner data,
  understand the scale of battery and recall issues, engage JLR at senior level, seek fair
  and consistent treatment, avoid fragmented complaints and unnecessary legal action where
  possible, and protect owner confidence and vehicle value.

## FAQ Detail

Include at least:

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

## Privacy and Terms Detail

Privacy should explain what data may be collected, why it is collected, how anonymised
statistics may be used, that personal data should not be published publicly, that Join
membership submissions and signed-in vehicle basics are stored with Netlify Functions and
Postgres, that Blobs are reserved for uploaded evidence files, that full VINs are not
stored, that uploads are not implemented in the MVP, that Netlify Identity is used for
authentication, and that a more formal privacy policy is needed before broader live
evidence data collection.

Terms / Participation Statement should say the group is informal unless a formal organisation
is created, the site does not provide legal advice, submitting data does not create a client
relationship with any lawyer, owners should submit accurate information to the best of their
knowledge, and data may be used in anonymised aggregate reporting if consent is given.

## Content constraints

- Do not include real owner data.
- Do not include unverifiable statistics as real figures.
- Mark placeholder figures and illustrative examples clearly.
- Do not use Jaguar/JLR logos, badges, readable plates, copyrighted press imagery, or other unlicensed manufacturer assets.
- If using a vehicle image, make it original/licensed/generated, avoid visible brand marks, and provide meaningful alt text.
- Keep public calls to action clear but not coercive.
- Include simple social share links for common channels such as X, Facebook, LinkedIn, and
  WhatsApp. Use accessible link text, open external share targets safely with
  `rel="noopener noreferrer"`, and keep the UI restrained.

## Validation

- Run `npm run build`.
- Check that every page has front matter with `layout`, `title`, and `description`.
- Check links among Join, Submit Vehicle Data, Evidence Dashboard, FAQ, Methodology, Privacy, and Terms.
