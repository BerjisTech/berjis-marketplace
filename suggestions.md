# Marketplace Suggestions

## Document gift card API in OpenAPI
- **Description:** The marketplace service exposes gift card CRUD endpoints, but they are not represented in `marketplace/service/openapi/marketplace.v1.yaml`. Adding the paths and schemas will help keep generated clients in sync and avoid regressions.
- **Tasks:**
  - [ ] Add `GET/POST/PATCH /v1/my/shops/{slug}/gift-cards` and transaction/report endpoints to the OpenAPI spec.
  - [ ] Regenerate any dependent SDKs or documentation that rely on the OpenAPI file.
  - [ ] Verify the updated spec aligns with current request/response payloads (including auto-generated codes).

## Allow manual orders for customers without linked user accounts
- **Description:** Manual order creation currently requires customers to have an associated platform user UUID. Support for guest checkouts or purely CRM-managed customers would let staff record phone or POS orders without forcing an account link.
- **Tasks:**
  - [ ] Update the marketplace service orders schema/logic to allow storing manual orders for customers without a user UUID.
  - [ ] Adjust customer lookups and reporting queries to fall back to customer UUID/email when user UUID is absent.
  - [ ] Extend the Angular manual order flow to surface guest customers while keeping the appropriate warnings.

## Restock inventory on order cancellation
- **Description:** Cancelling an order currently reverses discounts and gift card usage but does not adjust inventory levels. Automatically restocking cancelled quantities would keep stock figures accurate when orders are voided.
- **Tasks:**
  - [ ] Update the cancellation handler to increment inventory/adjustment records for each order item.
  - [ ] Ensure inventory alerts and low-stock signals respect the restock.
  - [ ] Surface the restock action in dashboards/audit logs so staff can trace the change.

## Show staff names in order timeline
- **Description:** Timeline events display truncated UUIDs for staff actions. Surfacing the staff member's display name would give better context to merchants reviewing activity.
- **Tasks:**
  - [ ] Fetch staff profile metadata (name/avatar) for the UUIDs referenced in timeline events.
  - [ ] Extend the marketplace service to include actor display info in `/v1/orders/:id/events` responses or provide a batch lookup endpoint.
  - [ ] Update the Angular dashboard timeline to render the resolved display names and avatars where available.

## Replace refund prompt with guided modal
- **Description:** Partial refunds rely on a `window.prompt`, which is brittle and blocks additional validation. A dedicated modal should collect the amount, reason, and optional communication to improve UX.
- **Tasks:**
  - [ ] Create a refund modal component with amount input, validation, and reason dropdown/textarea.
  - [ ] Reuse the modal for full and partial refunds, replacing the current prompt-based flow.
  - [ ] Ensure the modal invokes the existing refund API and updates the timeline/orders list upon success.

## Support clearing draft expiration dates
- **Description:** Draft order updates can set a new expiration date but cannot clear an existing one because the API treats missing values as no-op. Allowing drafts to revert to no expiry would simplify long-lived drafts.
- **Tasks:**
  - [ ] Adjust the draft update handler to distinguish between omitted fields and explicit nulls (e.g., use pointer wrappers).
  - [ ] Update the Angular draft editor to send a null payload when merchants remove the expiration date.
  - [ ] Add integration tests to confirm clearing and setting expirations behave as expected.

## Automate abandoned checkout outreach
- **Description:** We surface abandoned checkouts but merchants still need to contact customers manually. Automating reminder emails would improve recovery rates.
- **Tasks:**
  - [ ] Add an async job that sends branded reminder emails when a checkout has been inactive for a configured window.
  - [ ] Provide admin toggles per shop to enable reminders and customize cadence/content.
  - [ ] Track follow-up status so the dashboard shows whether outreach has been attempted automatically.

## Extend fulfillment analytics visibility
- **Description:** The new fulfillment workflow exposes granular status but the Angular dashboards still summarize orders without highlighting fulfillment completion rates or bottlenecks. Adding analytics will help merchants understand pickup/shipping performance.
- **Tasks:**
  - [ ] Add fulfillment metrics (pending, ready, shipped, delivered counts and average lead time) to `/v1/my/shops/:slug/metrics`.
  - [ ] Display fulfillment summary cards in the admin dashboard with drill-down links.
  - [ ] Provide filters in the orders list to show orders awaiting fulfillment actions.

## Automate campaign delivery worker
- **Description:** Campaign emails are queued via `campaign_messages`, but no worker actually sends them. A background processor should deliver scheduled messages and update status/error columns.
- **Tasks:**
  - [ ] Add a service worker that polls `campaign_messages` for due items and sends email through the configured provider.
  - [ ] Update message records with send results, retries, and error info.
  - [ ] Surface delivery stats in the marketing dashboard and expose an admin endpoint for monitoring.

## Build attribution insights dashboard
- **Description:** We now log UTM and attribution data, but there’s no reporting UI summarising performance by source/medium or conversions. A dashboard would help marketing teams act on the captured data.
- **Tasks:**
  - [ ] Aggregate marketing_attributions + orders into daily source/medium stats.
  - [ ] Expose an API endpoint that returns attribution metrics for analytics widgets.
  - [ ] Update the marketing admin page to visualise conversions and top channels.
