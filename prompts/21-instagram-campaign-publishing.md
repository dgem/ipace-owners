# Instagram Campaign Publishing Prompt

Use this prompt when creating or publishing organic Instagram content for the owners' group.
It records the product boundary between creative chat work and the deliberate admin send tool.

## Workflow

1. Use chat to develop the concept, caption, accessibility notes, on-screen copy and final
   media-generation prompt.
2. Generate the actual final media with a suitable image or video tool. Chat output and a
   storyboard are not proof that a video asset matches the prompt.
3. Put the reviewed MP4 or MOV at a public site-relative path. Never accept an arbitrary
   external URL in the admin form.
4. An administrator watches the complete file, confirms the media-review statement, and asks
   the server to validate and preview the exact path and caption.
5. Previewing has no Instagram side effect. The server returns a deterministic campaign ID
   and exact `PUBLISH <digest>` confirmation derived from the normalized path and caption.
6. Publishing requires the same authenticated admin, campaign ID, path, caption, media-review
   confirmation and exact server-generated phrase. Any edit requires a new preview.
7. The Go Function creates a Meta Reel container, polls until processing finishes, publishes
   it, and returns only the Instagram media ID. Provider tokens and response bodies must never
   be returned to the browser or logged.

This is controlled organic publishing, not paid advertising, automated engagement, following,
commenting or direct messaging. Scheduling and paid promotion are future work and must not be
implied by the initial publish-now interface. Reserve the deterministic campaign ID in the
`instagramCampaigns` Firestore ledger before contacting Meta; completed IDs return the existing
media ID, while processing or failed IDs require operator verification rather than a blind retry.

## Launch Reel creative contract

Generate the final Reel in Google Flow with Veo 3.1 in its highest-quality mode and native audio.
Veo is preferred because this campaign needs physically coherent automotive motion, strong prompt
adherence, a portrait composition, synchronized environmental sound, a cinematic sound bed and
precisely timed sound effects. If Flow limits one generation to eight seconds, create the opening
shot first and use scene extension to reach the full 15-second duration while preserving the same
vehicle, camera path, road and audio continuity.

For programmatic generation, call Veo through the Vertex AI long-running video-generation API,
not through the Flow interface. Use the current production model ID
`veo-3.1-generate-001`; do not use the retired `veo-3.1-generate-preview` endpoint. Veo 3.1 API
clips are limited to 4, 6 or 8 seconds, so produce the 15-second Reel as a controlled two-stage
job:

1. Generate an eight-second 9:16, 1080p clip with native audio covering the continuous approach,
   progressive deceleration and controlled stop.
2. Extract the exact final frame and use it as the starting image for a second continuation clip.
   Hold the same stationary hero composition, preserve environmental audio, deliver the low
   cinematic resolution and then exactly two lock chirps with synchronized indicator flashes.
3. Join the clips server-side, remove any duplicated boundary frame or audio transient, trim the
   master to exactly 15 seconds, and encode it as H.264 video with AAC 48 kHz audio.
4. Store generated intermediates privately in Cloud Storage. Copy only the reviewed final master
   to the public campaign path. Generation completion must never trigger Instagram publication;
   the existing human review, preview and exact-confirmation workflow remains mandatory.

Run generation as an asynchronous job and persist the Vertex AI operation name and state. Never
hold an admin browser request open while polling. The generation service account needs only
Vertex AI invocation and access to the dedicated campaign-media bucket. Keep generated-video
operations and storage paths separate from member evidence and private member snapshots.

The first Reel is a 15-second, 9:16 native temporal video. It must contain continuous physical
motion rather than animated stills, keyframe dissolves, a slideshow, a Ken Burns treatment or a
morph between unrelated vehicle poses. A dark I-PACE drives purposefully along a quiet British
road at blue hour, approaches camera, decelerates smoothly and comes to a precise, dramatic but
completely normal and controlled stop. The stop represents owners coming together with purpose;
it is not caused by a fault, emergency or loss of control.

Preserve the same vehicle, road, weather, lighting and body details throughout. Motion must remain
physically coherent: natural wheel rotation, suspension movement, stable tyre contact, moving
reflections, changing road perspective and background parallax. Never teleport, morph, resize or
swap the vehicle. Use deep navy and restrained teal tones, wet-road reflections and premium
editorial automotive framing. Leave negative space above the vehicle for later editable text.

Generate synchronized sound with no dialogue. Begin with subdued wet-road tyre noise, restrained
electric-drivetrain sound, light wind and distant road ambience. Build a modern, premium
low-frequency cinematic pulse as the vehicle approaches, rising toward the stop without becoming
aggressive. Resolve with a clean, confident low impact as the car settles. After a short quiet
beat, play exactly two crisp vehicle-lock confirmation chirps — `beep, beep` — clearly separated
and synchronized with two subtle amber indicator flashes. They must sound like normal locking
confirmation, not an alarm, reversing alert, fault warning or emergency signal. Fade naturally
after the second chirp. Do not use copyrighted music, vocals, speech, sirens, horns or alarms.

Exclude breakdown cues, power loss, emergency braking, warning or hazard lights, smoke, skid
marks, collision, swerving, reckless speed, damage, emergency services, distressed occupants,
embedded logos, watermarks, generated text, distorted wheels or changing body panels.

Suggested message beats:

- 0–3 seconds: `I-PACE OWNERS / WORKING TOGETHER`
- 3–6 seconds: `Recalls. Battery faults. / Inconsistent outcomes.`
- 6–9 seconds: `NOT ANOTHER FORUM / An independent, organised voice.`
- 9–12 seconds: `CONSTRUCTIVE. EVIDENCE-LED. / Focused on fair outcomes.`
- 12–15 seconds: `ADD YOUR VOICE / ipace-owners.org / Free to join. Under a minute.`

`public/ipace-owners-instagram-launch-reel.mp4` is reserved for a future approved export. The
recovered keyframe-composite draft is not approved, must not be selected by default, and must not
be committed as preservation-critical media. Generate the native temporal video and synchronized
soundtrack defined below into private campaign storage, watch it in full, verify the stop cannot
be read as a fault, and obtain human approval before publishing or committing a public export.

## Copy-ready Veo 3.1 master prompt

Create a single continuous 15-second cinematic vertical video for an Instagram Reel, 9:16,
1080p, photorealistic premium automotive-advertising quality, with synchronized native audio.
This must be true temporal video with continuous physical motion, not animated still images,
keyframe dissolves, a slideshow, a Ken Burns effect or a morph between poses.

At blue hour on a quiet wet British country road, one dark Jaguar I-PACE drives purposefully
toward the camera at a controlled, legal speed. Begin with a low front three-quarter tracking
shot while the car is farther down the road. The camera moves smoothly backward as the vehicle
approaches. Show natural rotating wheels, stable tyre contact, subtle suspension travel, accurate
wet-road reflections, headlight reflections moving across the road, changing road perspective and
real background parallax. Maintain exactly the same vehicle identity, body proportions, dark
paint, wheel design, lighting and road environment for the entire shot.

The vehicle progressively and visibly decelerates. The camera eases with it. At approximately
eight seconds, the car comes to a precise, dramatic but completely normal and controlled stop in
a front three-quarter hero position. There is no fault, breakdown, emergency, sudden power loss
or loss of control. Hold the stationary hero composition for the remaining seconds, with generous
dark negative space above the vehicle for later editable Instagram text overlays.

Sound design synchronized to the action: begin with subdued wet-road tyre noise, restrained
electric-drivetrain sound, light wind and distant road ambience. Build a modern premium
low-frequency cinematic pulse as the car approaches, increasing energy without becoming
aggressive. Rise into the deceleration, then resolve at the stop with one clean, confident low
cinematic impact as the suspension settles. After a short quiet beat, play exactly two crisp
vehicle-lock confirmation chirps — beep, beep — clearly separated. Synchronize the two chirps with
two subtle amber indicator flashes. The chirps are friendly normal locking confirmation, not an
alarm, warning, reversing alert or emergency signal. Let the road ambience fade naturally after
the second chirp. No dialogue, voice-over, vocals, copyrighted song, siren, horn, alarm or warning
tone.

Visual mood: deep navy and restrained teal, wet asphalt reflections, credible British landscape,
calm confidence, constructive purpose and premium editorial cinematography. Keep the car fully in
its lane and under control. No people. No readable registration. No manufacturer badge or logo.
No embedded text or watermark. No smoke, tyre smoke, skid marks, hard emergency braking, hazard
lights, warning lights, collision, swerving, reckless speed, damaged bodywork, emergency services,
distressed occupants, distorted wheels, body-panel changes, teleporting, vehicle swapping,
unwanted camera cuts or identity drift.

## API and configuration contract

Generation routes are `POST /api/admin/instagram-generate` and
`POST /api/admin/instagram-generation-status`. Both require a server-verified Firebase admin claim and
origin checks. Starting generation requires a stable browser request ID and exact `GENERATE
VIDEO` confirmation. Reserve the request before calling Vertex, generate an eight-second 9:16
1080p native-audio opening, and use that MP4 as Veo 3.1 video-extension input for the supported
seven-second continuation. Transactionally claim the extension phase before the second billable
provider call so concurrent status polls cannot duplicate it; fail closed if a provider call or
subsequent ledger write is indeterminate. Promote the completed 15-second result into `masters/`;
never invoke publishing from generation. Store job state in `instagramGenerationJobs`.

Return completed private media through a short-lived `/api/instagram-media/**` bearer path. Store
only its token hash and expiry, compare in constant time, keep the GCS URI private, and support GET,
HEAD and a single HTTP byte range. The delivery URL may be fetched by the review video player and
Meta, but expiry must fail closed.

Publishing routes are `POST /api/admin/instagram-preview` and
`POST /api/admin/instagram-publish`. Both require a server-verified Firebase admin claim and origin
checks. Runtime configuration is optional and fail-closed:

- `INSTAGRAM_ACCESS_TOKEN` — secret; never commit or expose it;
- `INSTAGRAM_USER_ID` — professional Instagram account ID;
- `INSTAGRAM_GRAPH_API_VERSION` — explicitly selected supported version such as `vNN.0`;
- `INSTAGRAM_MEDIA_BASE_URL` — HTTPS public origin Meta can fetch, with
  `RESEND_ASSET_BASE_URL` as the existing compatible fallback.

OpenTofu owns the `instagram-access-token` secret container and runtime accessor binding, but not
the token version. Keep OAuth token bytes out of tfvars and OpenTofu state. Enable the conditional
GitHub deployment variables only after an operator has added a valid secret version.

Use Meta's official Instagram Login content-publishing permission. Keep token acquisition,
rotation, account connection and Meta App Review as explicit operator setup. The media must meet
Meta's current Reel format constraints and be publicly fetchable while the container processes.
The required Instagram Login OAuth scopes are `instagram_business_basic` and
`instagram_business_content_publish`. Treat the Meta app ID and app secret as OAuth connection
credentials, not as a publishing API key; keep the app secret server-side if an automated account
connection flow is added.

OpenTofu must enable `aiplatform.googleapis.com`, create a private campaign-media bucket with
public access prevention, explicitly provision the managed Vertex AI service identity, grant its
service-agent role and bucket-scoped object access, grant the Function runtime
`roles/aiplatform.user` and object access on that bucket, and provide `CAMPAIGN_MEDIA_BUCKET`,
`VEO_LOCATION` (default `us-central1`), and `VEO_MODEL_ID` (default
`veo-3.1-generate-001`) through environment-specific GitHub variables. The generation UI must state
that Veo processing occurs in the US while the Function and private media bucket remain in London.
Use `work/` for source clips, continuation clips, extracted frames, and rejected candidates with
bounded retention. Use the versioned, non-expiring `masters/` prefix for approved final masters.
Use runtime identity rather than an API key. Persist and return only safe, actionable provider
failure classifications plus provider code/status for administrator diagnosis. Completing
generation must not invoke publishing.

## Verification

Add Go tests for admin authorization, strict request decoding, media-path allowlisting,
caption limits, review confirmation, exact publish confirmation, configuration failure and
provider failure. Add browser contract tests for token use, preview invalidation and disabled
publish controls. Run `make test` and `make build`.
