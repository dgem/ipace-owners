# Design System and Accessibility Prompt

Refine the visual system and accessibility baseline for the I-PACE Owners' Advocacy Group website.

## Goal

Create a calm, credible, technical-feeling public site that is mobile-first, accessible, and easy to maintain without a CSS framework.

## CSS rules

- Put all CSS in `src/assets/css/site.css`.
- Do not add Tailwind, Bootstrap, utility-first frameworks, Sass, PostCSS, or a CSS build step.
- Structure the stylesheet in this order:
  1. Modern reset
  2. Design tokens
  3. Base typography
  4. Layout utilities
  5. Components
  6. Print styles
- Prefer reusable component classes over inline styles.
- Use CSS custom properties for color, spacing, typography, shadows, radii, layout widths, and transitions.
- Homepage hero media should be integrated as part of the first-viewport experience, not added as a disconnected decorative card.
- For brand/product/vehicle imagery, use licensed, original, or generated bitmap assets only; do not use manufacturer logos, badges, watermarks, or copyrighted press imagery.
- Favicon artwork should be simple, legible at small sizes, aligned with the navy/teal palette, and free of Jaguar/JLR logos or badges. It should read as an I-PACE-style low electric crossover silhouette rather than a generic upright car.
- Cookie/privacy notices should be compact, dismissible, keyboard accessible, and should not obscure primary form actions on common laptop viewports.
- Cookie/privacy notices must not depend on JavaScript for disclosure: include a no-JavaScript
  fallback with a privacy-policy link, while using JavaScript only to persist dismissal. The
  no-JavaScript fallback should sit in the page flow rather than permanently overlaying the
  viewport.

## Design tokens

Use this palette unless there is a deliberate product reason to change it:

- Background: `#f7f8fb`
- Surface: `#ffffff`
- Surface muted: `#eef2f7`
- Text: `#111827`
- Text muted: `#4b5563`
- Border: `#d9e1ea`
- Primary/navy: `#12324a`
- Primary dark: `#0b2233`
- Accent/teal: `#0f766e`
- Accent soft: `#ccfbf1`
- Warning/amber: `#b45309`
- Danger/red: `#b91c1c`
- Success/green: `#15803d`
- Focus outline: `#2563eb`

## Accessibility requirements

- Use semantic landmarks and heading order.
- Include a skip link.
- Ensure visible focus states on links, buttons, form controls, and nav items.
- Ensure color contrast meets WCAG AA.
- Respect `prefers-reduced-motion`.
- Do not rely on color alone to communicate meaning.
- Use labels, legends, fieldsets, hints, and alert regions for forms.
- Hidden form steps must not leave focusable controls in the tab order.
- Mobile navigation must expose correct `aria-expanded`, `aria-controls`, and labels.

## Components to support

- Header and responsive navigation.
- Mobile nav drawer.
- Footer.
- Hero.
- Hero media/image layout.
- Page header.
- Buttons and button variants.
- Cards.
- Callouts.
- Badges.
- Form controls.
- Progress indicators.
- Dashboard/stat/chart panels.
- Auth-gate panels.
- Cookie/privacy notice.

## Validation

- Run `make build`.
- Manually review narrow and desktop layouts.
- Check keyboard access for header navigation, mobile nav, forms, and auth-gated actions.
