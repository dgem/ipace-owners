# Member experiences, editorial review and impact

Build a member-experience workflow that turns personal experience into consented public advocacy and structured evidence.

## Member submission

- Only authenticated members with a saved name and at least one registered vehicle may submit.
- Use the member's non-unique display-name preference as the attribution while keeping contact and legal names private. If the preference is empty and a legal name is saved, default to its first name. Manage this once in My Data rather than asking for a separate attribution on each experience.
- Use a familiar, limited rich-text editor for bold, italics, lists and blockquotes, and store its 80–20,000 character content as safe Markdown. Do not accept arbitrary web links or HTML. Members may insert only images or videos already uploaded to that experience; store a stable `story-media:` reference in Markdown and render it as the corresponding uploaded media. Validate media references server-side. Do not ask the member for a title; derive an editorial title from the experience and allow admins to refine it.
- Require a linked vehicle, but show a vehicle chooser only when the member has more than one. Keep vehicle and optional evidence links after the main writing and media controls. Allow links to the member's existing service/fault, Executive Office and SoH records. References must be ownership-checked server-side.
- Encourage members to complete missing vehicle and history records so their story strengthens aggregate evidence.
- Accept up to five JPEG/PNG/WebP images and two MP4/QuickTime/WebM short videos, with a combined 500 MB limit. Resize oversized images in the browser before upload. Use a native multiple-file picker that can open device photo libraries or files, and show removable thumbnails before upload.
- Let members insert a selected local image or video into the editor before saving. Provide button controls to move inserted media between adjacent text blocks. Use a temporary browser preview until the upload completes, then replace it with the verified `story-media:` reference in the saved Markdown.
- Keep uploaded media removable while editing, including media on an already submitted story. Show existing and newly selected media with the same thumbnail card, filename and insert/remove actions; send the retained media IDs when saving so removal persists.
- Preserve the draft ID after its first save and reuse completed uploads on retry; prevent concurrent saves while retaining form fields in the final request, and include newly uploaded media IDs in the final save. Upload media directly to a private Cloud Storage bucket through short-lived, authenticated signed URLs. Verify size and content type before promoting a pending upload.
- Require confirmation that the account is truthful to the member's knowledge and that they own or may supply uploaded media.
- Record separate, versioned permission for publication on this website and for third-party social, press or publisher use.

## Review and publication

- Store drafts and submitted stories privately in Firestore.
- Use the lifecycle: draft, submitted, needs-member-clarification, editing, approved and published.
- Admins may edit the title, Markdown and attribution, keep private review notes, and send a separate clarification message visible to the member. Members may reopen drafts, submitted/in-progress stories and clarification requests, see the editorial sentiment summary, then resubmit them. Never expose private admin notes or detailed editorial analysis through the member API.
- Retain story, consent and editorial revisions so changes can be audited.
- AI analysis is admin-triggered only. Store the prompt version, model and result. It suggests grammar, structure and keywords, identifies sentiment, claims needing evidence, and possible privacy, safety or defamation concerns. It must preserve the author's language and never publish automatically.
- Public APIs return only published, website-consented fields and short-lived media URLs. Never expose member UID, email, private name, review notes or private model analysis.
- Public pages show the member's story with an anonymised summary of the linked evidence and links inviting readers to register or add their own vehicle records.
- Make the public story catalogue discoverable from the main navigation as “Owners’ stories” in both launch and full presentation modes; use the same title for the catalogue page and its story context.
- Show a short, plain-text first meaningful paragraph on each public story card, omitting media references and formatting markup, and link to the full experience. Render inline media where it is referenced in the story, then show a thumbnail gallery for all attached media below the story. Keep the public experience grid and the editor wide on desktop, and use the same rounded green action-button treatment for member actions.
- Give each published story a stable `/member-stories/<slug>/` URL, while retaining query-string links as a compatibility path. Firebase Hosting rewrites story paths directly to `/member-stories/index.html`; Eleventy’s development server uses middleware to serve the catalogue page for local slug routes too. Deployment smoke checks both the catalogue and a representative trailing-slash story URL. The client resolves the final path segment against the published story list. Members can see each submission's current review state, clarification message and published date/link.
- The public story share control opens working share actions for Facebook, X, WhatsApp and LinkedIn, plus copy-link actions for Instagram and TikTok and the native device share sheet where supported. Platform actions are tracked as consent-gated share-link clicks; they do not claim that a post was completed.
- The admin review workspace keeps the selected story open after a save and supports bulk archive, approval and publication actions for selected stories. Bulk publication applies the same approval and consent validation as the individual action.
- Call the member submission area “Member experiences” and the public catalogue “Owners’ stories”. Keep explanatory copy compact on mobile, and link directly back to My Data.
- Show a concise “What you can do next” callout on both My Data pages explaining why experiences matter, with a direct “Share your experience” action. Keep it in one horizontal row when desktop space permits and compact on mobile. Repeat that action beside Vehicle history so members can move from recording events to describing their impact.
- Make member callouts reusable, versioned and dismissible. Store dismissals against the authenticated account so they follow the member across devices; use a survey-specific ID so a later survey can appear even when an earlier one was dismissed.
- Include the member's experience text, consent, evidence counts and safe media metadata in both CSV and workbook account exports. Exclude contact details, storage object names, admin notes and AI analysis.

## Impact measurement

- Ask separately for optional story-analytics consent.
- After consent, create a random pseudonymous reader identifier in browser storage and record story views, engagement, CTA clicks and share actions with campaign/referrer and platform context. Admin analysis reports share-link clicks by platform, not confirmed posts or impressions.
- Do not use story analytics for advertising. Do not store raw IP addresses in the analytics record.
- Campaign-specific links should make it possible to compare which story and message led to registrations, survey participation or evidence contributions.

## Security and infrastructure

- Verify Firebase ID tokens server-side for member APIs and admin claims for editorial APIs.
- Resolve member-facing feature access through an authenticated API and enforce the same rule on the protected feature APIs. Support `all`, `off`, `admin`, and future Firebase role names. Staging exposes member experiences to all members; production initially exposes authoring only to admins and the `member-experiences-preview` role.
- Permit `blob:` only for CSP image and media sources so selected local files can be previewed before upload; do not allow blob scripts or a general blob default source.
- Use a dedicated private, versioned member-story media bucket with public access prevention and CORS limited to configured site origins.
- Give the runtime service account object access and self `signBlob` permission for signed URLs; pass its email as the non-secret `STORAGE_SIGNING_SERVICE_ACCOUNT` runtime setting and do not store a signing key.
- Clean abandoned pending uploads automatically. Keep published/original objects private and deliver them through expiring URLs.
- Add tests for eligibility, record ownership, consent gates, media limits, admin authorization, public-field filtering and event validation.

- The opt-in staging sign-in browser journey can also upload a real small MP4 to a dedicated staging member's reusable private draft. It checks the saved reference and signed video retrieval; the member needs a name, display name and story access; the test creates a synthetic staging vehicle if none exists. Require an explicitly configured staging origin, never submit the draft, and document cleanup of detached test objects in the private versioned bucket.

- Let a member soft-delete any owned experience after typing DELETE. Hide it from member lists and exports and from public output, retain it for admin recovery, and provide an admin deleted status filter and a typed-confirmation soft-delete action in the admin review panel. Deleted stories cannot be changed or published through review actions. Guard writes against concurrent changes so a stale edit cannot undo deletion.

### Planned member departure workflow

A member must be able to request departure, with an administrator approving the request before any
permanent deletion. Approved departure must delete all the member's records, including soft-deleted
stories, all generations of story media, profile and representation history, vehicles, SoH and
service records, survey responses, membership and contact records, private snapshots, and the
Firebase Auth account. The implementation must inventory connected providers and backups, prevent
concurrent writes or re-creation during deletion, fail closed and support safe retry of partial work.
Do not describe the request as complete until every applicable store has been addressed. This is a
planned requirement; the account departure workflow has not yet been implemented.

- The admin story list opens a separate full-width review and editing view, with a back control returning to the same filters and page after confirmation if edits are unsaved. Keep selection of table checkboxes separate from opening a story.

- Members with existing experiences see their list first and open either an existing experience or a new one in a separate editor; members without experiences start directly in the editor. Provide a back control to the list, confirm before discarding unsaved member edits, and preserve the separate admin review view.
