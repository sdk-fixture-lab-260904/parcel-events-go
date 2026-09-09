---
title: "github.com/sdk-fixture-lab-260904/parcel-events-go API"
sdk_target: "go"
---
<!-- managed by Doctorine — edits will be overwritten -->

# github.com/sdk-fixture-lab-260904/parcel-events-go API

Generated from the same resolved API model as the SDK.

## Operations

### POST /v1/labels

- SDK method: `labels.create`
- Operation: `post /v1/labels`

### GET /v1/labels/{label_id}

- SDK method: `labels.retrieve`
- Operation: `get /v1/labels/{}`

### DELETE /v1/pickups/{pickup_id}

- SDK method: `pickups.cancel`
- Operation: `delete /v1/pickups/{}`

### POST /v1/pickups

- SDK method: `pickups.create`
- Operation: `post /v1/pickups`

### GET /v1/pickups

- SDK method: `pickups.list`
- Operation: `get /v1/pickups`

### GET /v1/pickups/{pickup_id}

- SDK method: `pickups.retrieve`
- Operation: `get /v1/pickups/{}`

### POST /v1/rates/quote

- SDK method: `rates.quote`
- Operation: `post /v1/rates/quote`

### DELETE /v1/shipments/{shipment_id}

- SDK method: `shipments.cancel`
- Operation: `delete /v1/shipments/{}`

### POST /v1/shipments

- SDK method: `shipments.create`
- Operation: `post /v1/shipments`

### GET /v1/shipments/{shipment_id}/events

- SDK method: `shipments.events.list`
- Operation: `get /v1/shipments/{}/events`

### GET /v1/shipments

- SDK method: `shipments.list`
- Operation: `get /v1/shipments`

### GET /v1/shipments/{shipment_id}

- SDK method: `shipments.retrieve`
- Operation: `get /v1/shipments/{}`

### PATCH /v1/shipments/{shipment_id}

- SDK method: `shipments.update`
- Operation: `patch /v1/shipments/{}`

### GET /v1/tracking/{tracking_number}

- SDK method: `tracking.retrieve`
- Operation: `get /v1/tracking/{}`

### POST /v1/webhook-endpoints

- SDK method: `webhook_endpoints.create`
- Operation: `post /v1/webhook-endpoints`

### GET /v1/webhook-endpoints

- SDK method: `webhook_endpoints.list`
- Operation: `get /v1/webhook-endpoints`

### DELETE /v1/webhook-endpoints/{endpoint_id}

- SDK method: `webhook_endpoints.remove`
- Operation: `delete /v1/webhook-endpoints/{}`

### PATCH /v1/webhook-endpoints/{endpoint_id}

- SDK method: `webhook_endpoints.update`
- Operation: `patch /v1/webhook-endpoints/{}`
