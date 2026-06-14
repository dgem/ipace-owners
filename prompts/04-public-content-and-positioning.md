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
- Privacy: placeholder until formal review before live data collection.
- Participation statement/terms: non-legal-action participation framing.

## Content constraints

- Do not include real owner data.
- Do not include unverifiable statistics as real figures.
- Mark placeholder figures and illustrative examples clearly.
- Do not use Jaguar/JLR logos, badges, readable plates, copyrighted press imagery, or other unlicensed manufacturer assets.
- If using a vehicle image, make it original/licensed/generated, avoid visible brand marks, and provide meaningful alt text.
- Keep public calls to action clear but not coercive.

## Validation

- Run `npm run build`.
- Check that every page has front matter with `layout`, `title`, and `description`.
- Check links among Join, Submit Vehicle Data, Evidence Dashboard, FAQ, Methodology, Privacy, and Terms.
