# Member stories, editorial review and impact

Build a member-story workflow that turns personal experience into consented public advocacy and structured evidence.

## Member submission

- Only authenticated members with a saved name and at least one registered vehicle may submit.
- Use the saved member name as the default attribution while keeping contact details private.
- Accept a title and Markdown body of 80–20,000 characters, with prompts about what happened, its impact, the response received and the outcome sought.
- Require a linked vehicle. Allow links to the member's existing service/fault, Executive Office and SoH records. References must be ownership-checked server-side.
- Encourage members to complete missing vehicle and history records so their story strengthens aggregate evidence.
- Accept up to five JPEG/PNG/WebP images and two MP4/QuickTime/WebM short videos, with a combined 250 MB limit.
- Upload media directly to a private Cloud Storage bucket through short-lived, authenticated signed URLs. Verify size and content type before promoting a pending upload.
- Require confirmation that the account is truthful to the member's knowledge and that they own or may supply uploaded media.
- Record separate, versioned permission for publication on this website and for third-party social, press or publisher use.

## Review and publication

- Store drafts and submitted stories privately in Firestore.
- Use the lifecycle: draft, submitted, needs-member-clarification, editing, approved and published.
- Admins may edit the title, Markdown and attribution, keep private review notes, and send a separate clarification message visible to the member. Members may reopen drafts and clarification requests, then resubmit them. Never expose private admin notes through the member API.
- Retain story, consent and editorial revisions so changes can be audited.
- AI analysis is admin-triggered only. Store the prompt version, model and result. It suggests grammar, structure and keywords, identifies sentiment, claims needing evidence, and possible privacy, safety or defamation concerns. It must preserve the author's language and never publish automatically.
- Public APIs return only published, website-consented fields and short-lived media URLs. Never expose member UID, email, private name, review notes or private model analysis.
- Public pages show the member's story with an anonymised summary of the linked evidence and links inviting readers to register or add their own vehicle records.

## Impact measurement

- Ask separately for optional story-analytics consent.
- After consent, create a random pseudonymous reader identifier in browser storage and record story views, engagement, CTA clicks and share actions with campaign/referrer context.
- Do not use story analytics for advertising. Do not store raw IP addresses in the analytics record.
- Campaign-specific links should make it possible to compare which story and message led to registrations, survey participation or evidence contributions.

## Security and infrastructure

- Verify Firebase ID tokens server-side for member APIs and admin claims for editorial APIs.
- Use a dedicated private, versioned member-story media bucket with public access prevention and CORS limited to configured site origins.
- Give the runtime service account object access and self `signBlob` permission for signed URLs; do not store a signing key.
- Clean abandoned pending uploads automatically. Keep published/original objects private and deliver them through expiring URLs.
- Add tests for eligibility, record ownership, consent gates, media limits, admin authorization, public-field filtering and event validation.
