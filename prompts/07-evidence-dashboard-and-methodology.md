# Evidence Dashboard and Methodology Prompt

Implement or refine the public evidence dashboard and methodology pages.

## Goal

Show truthful, current aggregates from collected owner evidence without presenting
illustrative figures as real data.

## Evidence dashboard

- Present anonymised aggregate statistics only.
- Read the generated public statistics snapshot through `GET /api/public-stats`; do not
  query canonical member records from the browser.
- Show only metrics supported by fields currently collected. Use a clear "Not collected"
  state where a value has no eligible observations, and do not render sample percentages.
- Include owners contributing data, cars registered, latest average reported SoH, total SoH
  readings, cars with repeat readings, and average first-to-latest SoH change.
- Include a visible disclaimer explaining:
  - voluntary submission bias,
  - verification levels,
  - anonymisation,
  - aggregation,
  - exclusions from public statistics where needed.
- Include summary statistic cards.
- Use the deduplicated Join-submission total for the public “Owners joined” counter. Treat email
  case and `+tag` addressing as one identity. Keep this distinct from verified Firebase Auth
  accounts and never expose either set of addresses.
- Include chart-like panels for currently collected distributions:
  - latest State of Health per car;
  - registered cars by model year.
- Add future panels only when their source fields are persisted, reviewed, and included in
  the public aggregate snapshot, such as:
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

- Run `make build`.
- Confirm no real private data appears in static output.
- Confirm no illustrative numbers remain in live statistic or chart elements.
