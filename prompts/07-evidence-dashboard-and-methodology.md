# Evidence Dashboard and Methodology Prompt

Implement or refine the public evidence dashboard and methodology pages.

## Goal

Show how owner evidence will be presented once collection is live, without pretending placeholder figures are real.

## Evidence dashboard

- Present anonymised aggregate statistics only.
- Mark all placeholder figures as illustrative or sample data.
- Include a visible disclaimer explaining:
  - voluntary submission bias,
  - verification levels,
  - anonymisation,
  - aggregation,
  - exclusions from public statistics where needed.
- Include summary statistic cards.
- Include chart-like panels for distributions such as:
  - State of Health,
  - model year,
  - HV battery work by model year,
  - modules replaced,
  - recall completion,
  - inside vs outside 8-year / 100,000-mile battery warranty,
  - average time off road,
  - loan car provision,
  - payment/goodwill outcomes,
  - warranty status at time of fault,
  - HV battery work type.
- Use accessible HTML/CSS chart representations with text labels and `aria-label` where needed.
- Link to the methodology page.
- Invite users to join and submit vehicle data.

## Methodology page

Explain:

- What data owners submit.
- How verification levels work.
- How duplicates will be detected.
- How raw personal data and VINs are protected.
- How public statistics are anonymised and aggregated.
- Why sample bias matters.
- Why structured data is more credible than forum anecdotes.
- How excluded or uncertain records are handled.

## Verification levels

Use a clear model such as:

- Self-reported.
- Document supplied.
- Reviewed.
- Duplicate-checked.
- Excluded from public statistics.

## Validation

- Run `npm run build`.
- Confirm no real private data appears in static output.
- Confirm every placeholder number is clearly labelled.
