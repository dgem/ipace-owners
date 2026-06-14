# Original Project Prompt

This file stores the original prompt that defined the I-PACE Owners' Advocacy Group website MVP.

---

## Project prompt

Project: I-PACE Owners' Advocacy Group website.

Goal: build a static, mobile-first, accessible public website with Markdown-authored content, Netlify Identity login, gated placeholder member/admin pages, and multi-step data collection forms. This first PR should establish the site architecture, styling, public pages, content model, Netlify Identity integration, and frontend form UX. Do not implement real backend persistence yet.

Preferred stack:

* Eleventy / 11ty static site generator.
* Author public content pages in Markdown with front matter.
* Generate semantic, responsive, mobile-first HTML.
* Use custom CSS, not Tailwind.
* Include a modern CSS reset / compatibility baseline.
* Use CSS custom properties for colour, spacing, typography, shadows and radii.
* Use plain JavaScript only where useful:
    * mobile menu toggle
    * Netlify Identity UI state
    * multi-step form navigation
    * client-side form progress indicators
    * light validation helpers
* Target Netlify deployment.
* Add Netlify Identity in this PR.
* Prepare for future Netlify Functions and Netlify Blobs, but do not implement storage yet.

Important implementation choices:

* Do not use Tailwind.
* Do not use Bootstrap.
* Do not use React, Vue, Svelte or another heavy frontend framework.
* Do not implement a database layer in this PR.
* Do not persist submitted data in Git.
* Do not store form submissions locally except perhaps temporary in-memory/client-side state while navigating the form.
* Do not include real owner data.
* Do not use Jaguar/JLR logos or copyrighted imagery unless already licensed.
* Keep HTML semantic and accessible.

NPM / NPX expectations:

* Use npm scripts for repeatable project commands: npm run dev, npm run build, npm run clean.
* Include a package-lock.json.

Tone and positioning:

* The group should sound constructive, evidence-led, collaborative and serious.
* The site should make clear that the aim is to work positively with JLR where possible, while organising owners so there is credible data and a collective voice.
* Avoid aggressive legal language on the homepage.
* Legal action should be framed only as a possible alternative that a constructive resolution may help avoid.
* Include the phrase "Speak softly and carry a big stick" somewhere appropriate, probably on the About page, not as the main homepage slogan.
* Do not make claims that the group is already formally incorporated or legally represented.
* Do not imply that submitted data is legally verified.

Recommended colour scheme:
Use a calm, credible, technical-feeling palette. Define it in CSS custom properties.

Suggested palette:

* Background: #f7f8fb
* Surface: #ffffff
* Surface muted: #eef2f7
* Text: #111827
* Text muted: #4b5563
* Border: #d9e1ea
* Primary/navy: #12324a
* Primary dark: #0b2233
* Accent/teal: #0f766e
* Accent soft: #ccfbf1
* Warning/amber: #b45309
* Danger/red: #b91c1c
* Success/green: #15803d
* Focus outline: #2563eb

[Full prompt continues — see problem statement for complete requirements]
