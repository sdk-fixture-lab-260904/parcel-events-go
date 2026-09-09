<!-- managed by Doctorine — edits will be overwritten -->
# Pagination

## cursor

Stop when the page has zero items OR the next-cursor field is null/empty.

- MUST treat the cursor/token value as OPAQUE — pass it through to the next request UNCHANGED; never parse, decode, increment, or reconstruct it.
- If the very first page returns zero items, stop immediately; do not issue a second request.

| Operation | Request bindings | Response bindings |
| --- | --- | --- |
| <a id="listpickups"></a>`GET /v1/pickups` | `cursor` → query.cursor · `limit` → query.limit | `items` → $.data · `next_cursor` → $.next_cursor |
| <a id="listshipments"></a>`GET /v1/shipments` | `cursor` → query.cursor · `limit` → query.limit | `items` → $.data · `next_cursor` → $.next_cursor |
| <a id="listshipmentevents"></a>`GET /v1/shipments/{shipment_id}/events` | `cursor` → query.cursor · `limit` → query.limit | `items` → $.data · `next_cursor` → $.next_cursor |
| <a id="listwebhookendpoints"></a>`GET /v1/webhook-endpoints` | `cursor` → query.cursor · `limit` → query.limit | `items` → $.data · `next_cursor` → $.next_cursor |
