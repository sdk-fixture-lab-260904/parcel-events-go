---
name: "parcel-events-go"
description: "Integrate the Parcel Events Sandbox API HTTP API: labels, pickups, rates, shipments, tracking, webhook endpoints. Use when writing code that calls Parcel Events Sandbox API, or when the user mentions labels, pickups, rates, shipments, tracking, webhook endpoints."
---
<!-- managed by Doctorine — edits will be overwritten -->

# parcel-events-go

Integrate the Parcel Events Sandbox API HTTP API: labels, pickups, rates, shipments, tracking, webhook endpoints. Use when writing code that calls Parcel Events Sandbox API, or when the user mentions labels, pickups, rates, shipments, tracking, webhook endpoints.

## Authentication

Default scheme (`api_key`):

- Send `Parcel-API-Key: <value>` as a request header — value from `PARCEL_EVENTS_API_KEY`.

Escape hatches (other accepted schemes):
- `bearer_token` (bearer): Send `authorization: Bearer <value>` — value from `PARCEL_EVENTS_BEARER_TOKEN`.

## Base URL

This API models 2 environments — decide which one to call:

- If targeting **production** (default), use `https://api.parcel-events.example`.
- If targeting **sandbox**, use `https://sandbox.parcel-events.example`.

## Errors

- `AuthenticationError` (401) — branch on status, never substring-match the body.
- `ConflictError` (409) — branch on status, never substring-match the body.
- `NotFoundError` (404) — branch on status, never substring-match the body.
- `PaymentRequiredError` (402) — branch on status, never substring-match the body.
- Retry 408, 409, 429, 5XX with backoff; honor `Retry-After`.
Full status→action table: [reference/errors.md](reference/errors.md).

## Pagination

Schemes used: cursor.

Termination predicates + a verified loop per scheme: [reference/pagination.md](reference/pagination.md).

## Client recipe

Read [recipes/http-client.md](recipes/http-client.md) before writing a client.

## Operations

- POST /v1/labels · createLabel · reference/labels.md#createlabel `body required`
- GET /v1/labels/{label_id} · retrieveLabel · reference/labels.md#retrievelabel
- DELETE /v1/pickups/{pickup_id} · cancelPickup · reference/pickups.md#cancelpickup
- POST /v1/pickups · createPickup · reference/pickups.md#createpickup `body required`
- GET /v1/pickups · listPickups · reference/pickups.md#listpickups
- GET /v1/pickups/{pickup_id} · retrievePickup · reference/pickups.md#retrievepickup
- POST /v1/rates/quote · quoteRates · reference/rates.md#quoterates `body required`
- DELETE /v1/shipments/{shipment_id} · cancelShipment · reference/shipments.md#cancelshipment
- POST /v1/shipments · createShipment · reference/shipments.md#createshipment `body required`
- GET /v1/shipments · listShipments · reference/shipments.md#listshipments
- GET /v1/shipments/{shipment_id} · retrieveShipment · reference/shipments.md#retrieveshipment
- PATCH /v1/shipments/{shipment_id} · updateShipment · reference/shipments-2.md#updateshipment `body required`
- GET /v1/shipments/{shipment_id}/events · listShipmentEvents · reference/shipments.md#listshipmentevents
- GET /v1/tracking/{tracking_number} · retrieveTrackingTimeline · reference/tracking.md#retrievetrackingtimeline
- POST /v1/webhook-endpoints · createWebhookEndpoint · reference/webhook-endpoints.md#createwebhookendpoint `body required`
- GET /v1/webhook-endpoints · listWebhookEndpoints · reference/webhook-endpoints.md#listwebhookendpoints
- DELETE /v1/webhook-endpoints/{endpoint_id} · deleteWebhookEndpoint · reference/webhook-endpoints.md#deletewebhookendpoint
- PATCH /v1/webhook-endpoints/{endpoint_id} · updateWebhookEndpoint · reference/webhook-endpoints.md#updatewebhookendpoint `body required`

## Protocol

Open ONLY the `reference/<resource>.md` file for the resource you need;
`openapi/openapi.json` is the schema truth for anything not answered in markdown.
