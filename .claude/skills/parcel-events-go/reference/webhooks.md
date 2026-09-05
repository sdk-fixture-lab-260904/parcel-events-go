<!-- managed by Doctorine — edits will be overwritten -->
# Webhook events

Inbound events this API delivers TO your endpoint — documented for coverage; they are NOT callable operations.

| Event | Payload | Summary |
| --- | --- | --- |
| `shipmentStatusChanged` | `ShipmentStatusEvent` | Shipment status changed |

## Delivery authenticity

The spec does not declare a webhook signature scheme the SDK can verify against, so it ships NO verifier. Authenticate deliveries per the API provider’s own docs BEFORE trusting a parsed body.

- MUST NOT assume a `sha256=<hex>` HMAC header — that convention is knowable only for GitHub-style providers.
- MUST process the unparsed raw body for any authenticity check (key order and whitespace change digests).
