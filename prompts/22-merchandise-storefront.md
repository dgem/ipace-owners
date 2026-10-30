# Public merchandise storefront

## Product decisions

Build `/shop/` for everyone, including guests. Do not require Firebase sign-in or membership,
create membership records, enrol shoppers in campaigns, or mix order data with vehicle evidence.
Use the site's Eleventy/Nunjucks templates, vanilla `merchandise.js` and custom CSS.

Use the existing Printify custom-integration shop for catalogue and fulfilment, and Stripe
hosted Checkout for one-off payments in GBP. Support international destinations through an
explicit configured destination list and a real Printify shipping quote for each basket/address.
Never advertise worldwide availability where a provider cannot deliver.

The shop is not intended to generate profit or revenue for its own sake. Keep prices as close
to cost as practicable. Any surplus, including rounding, contributes to site running costs,
currently around £30 per month. Do not describe purchases as charitable donations.

The user's price rule applies to every merchandise GBP price displayed or charged: round UP
to the smallest amount ending in `.49` or `.99`. Use integer pence. Internal cost components
need not end in those amounts and must not be presented as separately payable retail prices.
The final basket price is rounded once after production, shipping, configured supplier-tax
allowance and configured payment-cost recovery. No percentage profit margin is added.

## Implementation and privacy

Routes: `/shop/`, `/shop/terms/`; public navigation includes Shop.
The API router serves these endpoints through the existing `Api` function:

- `GET /api/shop/catalogue`: enabled/available variants, provider images, plain-text product
  descriptions and production prices; a one-minute in-process catalogue cache. No token,
  private order fields or provider HTML is exposed.
- `POST /api/shop/quote`: validate a maximum of ten distinct lines, ten units per line and
  twenty units total, address, supported country, terms and honeypot. Re-fetch each product
  and quote standard shipping with Printify. Store a private quote, return its rounded total
  and a cryptographically random 256-bit receipt. Reject unknown JSON fields/oversized bodies.
- `POST /api/shop/checkout`: accept the receipt in `Authorization: Bearer`; use only persisted
  quote details and total. Create a GBP, card-only Stripe Checkout Session with an order-specific
  idempotency key and a fixed one-hour expiry. Quote initiation expires after 30 minutes.
  Browser-supplied totals, product descriptions and redirect URLs are not trusted.
- `GET /api/shop/order`: receipt-authorised status, item summary, rounded total and reference.
  Do not return email, delivery address, provider credentials or other shoppers' orders.
- `POST /api/shop/stripe-webhook`: raw-body HMAC validation, multiple v1 signature support,
  five-minute timestamp tolerance; only paid Checkout events can trigger fulfilment. Verify
  stored session ID, order ID, GBP total/currency and shop before claiming in a transaction.
- `POST /api/shop/printify-webhook`: raw-body `X-Pfy-Signature` HMAC, shop validation, then
  fetch the current provider status instead of trusting potentially stale event state.

`merchandiseOrders` is private Firestore data. Documents contain shop/order identifiers,
line items, delivery contact/address, internal integer cost components, summary, creation/quote
expiry, terms version, Stripe session ID/URL, Printify ID and status. The document ID is a SHA-256
hash of the random receipt; the raw receipt is never stored server-side. Browser session storage
keys are scoped by order ID. Losing the browser receipt does not expose an order by guessing its
public reference; email/contact support is the fallback. Do not add these records to snapshots,
public aggregate exports or membership CSVs. Do not log request bodies, receipt secrets or PII.

Persist `submitting` before creating a Printify order. Persist the returned Printify ID before
requesting production. Retries with a known ID can safely inspect/resume that same order; never
create another. A timeout, crash or ambiguous response before recording the ID leaves
`submitting`/`needs-review` and must be reconciled against Printify's external order ID by an
operator. Do not assume `external_id` offers undocumented idempotency. Log a non-PII operational
error and return a retryable webhook response. Payment success is not inferred from a redirect.

## Configuration and release

`MERCH_CONFIG` is a non-secret JSON object containing:

- `enabled`, `shopId`, `baseUrl` (fixed HTTPS site origin).
- `costToGbpMicros` and `shippingToGbpMicros`: explicitly verified conversion factors in
  millionths, separately for API production costs and shipping. One source unit to one GBP is
  1000000. Printify catalogue, billing and retail currencies differ: never infer these factors
  from the account's display currency. Confirm using an actual quote/billing comparison.
- `paymentFeeBps`, `paymentFeePence`: reviewed payment-cost allowance, incorporated in the
  merchandise total rather than a card surcharge. Use the actual Stripe account/card mix.
- `reviewedUntil`: RFC3339 expiry for the reviewed pricing policy. Expiry stops new sales;
  already-paid orders and status webhooks continue to process using the shop configuration.
- `countries`: ISO country code keys mapping to `name` and `supplierTaxBps`; enable each
  destination only after its supplier taxes, local tax obligations and delivery have been
  reviewed. Tax allowance is not a calculation of the seller's VAT liability or customs duty.
- `sellerName`, `sellerAddress`, `termsReviewed`: seller identity is displayed before payment;
  checkout remains unavailable until these are supplied and shop terms are reviewed.

Provider credentials are `PRINTIFY_API_TOKEN` and `STRIPE_SECRET_KEY`. Keep them in the ignored
environment-scoped `.env`; never commit values or put them in browser assets, Functions JSON env
files, logs or OpenTofu state. Secret resources and runtime IAM are managed in OpenTofu; the local
bootstrap helper imports these API credentials into Secret Manager. During every `make infra-apply`,
OpenTofu's local reconciler creates or updates the environment's Stripe and Printify webhook
registrations and stores generated signing secrets directly in Secret Manager, never in OpenTofu
state or logs. The ignored
repo-root `.env` is loaded by `scripts/infra-env.sh` for `make infra-apply`; use `_STAGING` and
`_PRODUCTION` suffixes for every bootstrap credential, including the existing application
secrets `VIN_PEPPER`, `RESEND_API_KEY`, `IDEAL_POSTCODES_API_KEY`, and `INSTAGRAM_ACCESS_TOKEN`.
Each apply imports only matching values; unscoped values are removed and never used as fallback.
Preserve the existing production VIN pepper because it determines stored vehicle HMACs. OpenTofu's
`merchandise_config` and `merchandise_secrets_enabled` variables write the corresponding
`MERCH_CONFIG_STAGING/PRODUCTION` and `MERCH_SECRETS_ENABLED_STAGING/PRODUCTION` GitHub environment
variables. Secret bindings default off and must be enabled explicitly after secret versions exist.

At initial implementation, the user supplied a local Printify token and has a custom-integration
shop with a tote, T-shirt, car magnet and mug. The Stripe account's continued availability and
Printify API currency factors are not yet confirmed. Live checkout must remain disabled until
setup and end-to-end validation are complete. Never create sample orders as a read-only check.

Use a separate staging Printify shop and Stripe test keys; Printify order submission/production
can incur real charges even when a Stripe test payment is used. Disable fulfilment in staging
unless explicitly exercising it with approved test merchandise. Configure Stripe receipts and
order confirmation terms, and Printify shipment notifications before launch. Webhook endpoints
must use a stable environment origin, not an expiring preview URL.

OpenTofu registers Stripe `checkout.session.completed` and
`checkout.session.async_payment_succeeded`, plus Printify product, publication, order and shipment
events. Signing secrets are generated or received during the API calls and written directly to
the environment's Secret Manager versions. Ensure the Printify account has funding. Order
submission is followed by explicit send-to-production once the provider ID is saved, so
auto-approval settings are not relied on.

## Operations and acceptance

For an ambiguous submission, locate the order in Stripe and Printify by the full external ID.
If Printify has it, attach that provider ID to the private order and replay the signed Stripe
webhook. If it does not, confirm absence before resetting to `quoted` and replaying. Never reset
blindly. If fulfilment cannot proceed, refund through Stripe and cancel through Printify as
separate operations; record both in support notes. Monitor merchandise error logs and Stripe
failed webhook deliveries. Do not claim a refund happened merely because Printify cancelled.

Remove abandoned unpaid quotes after seven days only after checking they have no paid session.
Retain paid records only for fulfilment/returns/disputes and applicable accounting obligations;
handle customer access/deletion requests via order email, separately from membership exports.
Preserve Printify designs/source artwork, shop/product/variant configuration, provider webhook
registrations, Stripe account settings and secret recovery material outside the repository.

Add tests for guest access, receipt protection, malformed/oversized input, destination/quantity
validation, stale products, price tampering, GBP rounding boundaries, provider errors, signature
forgery/replay, unpaid events, duplicate/ambiguous order handling and persisted session matching.
Run `make test`, `make build`, `make dev` and `make test-visual`; inspect all desktop/mobile images,
including shop catalogue, basket/quote, unavailable checkout and order-status states.

`MERCH_CONFIG.stripeLive` selects test/live payments and must match the Stripe secret key prefix
and Checkout session livemode. Test payments are stored as `test-paid` and NEVER submitted to
Printify. The storefront prominently labels test mode. Only explicitly configured live payments
can create merchandise orders. Keep test/live Stripe endpoint signing secrets separate.

The committed `src/_data/merchandisePreview.json` and `public/images/merchandise/*.jpg` show the
four initial products when the live catalogue is unavailable. These cards can be added to a
local preview basket, but have no variants, prices, delivery form, or ordering action. The
browser must show a friendly preparation message instead of surfacing API errors. When the API
returns live catalogue products, those products are authoritative for availability/options/prices;
delivery fields and ordering stay hidden until `checkoutEnabled` is true. Preserve the source
Printify mockups and designs externally.

Subscribe the Printify webhook to product create/update/delete and `product:publish:started`
as well as order created/updated/cancelled/sent/shipments events. OpenTofu performs this
reconciliation through the Printify API using the configured shop ID and stable origin. Product
events invalidate the short catalogue cache. Verified publish
create/update requests re-read the configured shop product, then acknowledge publication with
its `/shop/#product-<id>` handle; unpublish requests use Printify's unpublish endpoint. This
completes custom-channel publication and releases the publishing lock. Existing locked products
need publication retriggered after the webhook is installed; do not change them during inspection.
