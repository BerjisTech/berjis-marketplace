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
