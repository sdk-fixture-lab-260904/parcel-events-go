<!-- managed by Doctorine — edits will be overwritten -->
## Contents

- [DELETE /v1/shipments/{shipment_id}](#cancelshipment)
- [POST /v1/shipments](#createshipment)
- [GET /v1/shipments/{shipment_id}/events](#listshipmentevents)
- [GET /v1/shipments](#listshipments)
- [GET /v1/shipments/{shipment_id}](#retrieveshipment)

# shipments

<a id="cancelshipment"></a>
### DELETE /v1/shipments/{shipment_id}

Cancel a shipment before handoff

`DELETE /v1/shipments/{shipment_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| shipment_id | path | string | yes |

Response `204`: no body.

<a id="createshipment"></a>
### POST /v1/shipments

Create a shipment

`POST /v1/shipments`

| name | in | type | required |
| --- | --- | --- | --- |
| Idempotency-Key | header | string | yes |

Request body (`application/json`): `CreateShipmentRequest` (required).

_Example (synthesized from schema):_

```json
{
  "delivery_instructions": "Leave with the synthetic reception desk",
  "metadata": {
    "additionalProp1": "string"
  },
  "parcels": [
    {
      "contents": [
        {
          "customs_code": "610910",
          "description": "Synthetic cotton shirt",
          "quantity": 2,
          "unit_value": {
            "amount": "12.50",
            "currency": "XTS"
          }
        }
      ],
      "height_cm": 12,
      "length_cm": 30.5,
      "weight_grams": 1750,
      "width_cm": 20
    }
  ],
  "recipient": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "reference": "synthetic-order-1042",
  "sender": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "service": "economy"
}
```

Response `201`: `Shipment`.

_Example (synthesized from schema):_

```json
{
  "created_at": "2026-09-04T08:15:30Z",
  "delivery_instructions": "string",
  "id": "shp_00000000000000000001",
  "metadata": {
    "additionalProp1": "string"
  },
  "object": "shipment",
  "parcels": [
    {
      "contents": [
        {
          "customs_code": "610910",
          "description": "Synthetic cotton shirt",
          "quantity": 2,
          "unit_value": {
            "amount": "12.50",
            "currency": "XTS"
          }
        }
      ],
      "height_cm": 12,
      "length_cm": 30.5,
      "weight_grams": 1750,
      "width_cm": 20
    }
  ],
  "recipient": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "reference": "synthetic-order-1042",
  "sender": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "service": "economy",
  "status": "canceled",
  "tracking_number": "PE-000000000001",
  "updated_at": "2026-09-04T08:16:12Z"
}
```

<a id="listshipmentevents"></a>
### GET /v1/shipments/{shipment_id}/events

List tracking events for a shipment

`GET /v1/shipments/{shipment_id}/events`

| name | in | type | required |
| --- | --- | --- | --- |
| shipment_id | path | string | yes |
| cursor | query | string | no |
| limit | query | integer | no |

Response `200`: `TrackingEventPage`.

_Example (synthesized from schema):_

```json
{
  "data": [
    {
      "id": "evt_00000000000000000001",
      "location": {
        "city": "Samplehaven",
        "country": "ZZ"
      },
      "message": "Parcel scanned at synthetic sorting hub",
      "object": "tracking_event",
      "occurred_at": "2026-09-05T05:40:00Z",
      "shipment_id": "shp_00000000000000000001",
      "status": "canceled"
    }
  ],
  "has_more": true,
  "next_cursor": "string"
}
```

Paginated (`cursor` scheme) — see [pagination.md](pagination.md#listshipmentevents).

<a id="listshipments"></a>
### GET /v1/shipments

List shipments

`GET /v1/shipments`

| name | in | type | required |
| --- | --- | --- | --- |
| cursor | query | string | no |
| limit | query | integer | no |
| status | query | ShipmentStatus | no |

Response `200`: `ShipmentPage`.

_Example (synthesized from schema):_

```json
{
  "data": [
    {
      "created_at": "2026-09-04T08:15:30Z",
      "delivery_instructions": "string",
      "id": "shp_00000000000000000001",
      "metadata": {
        "additionalProp1": "string"
      },
      "object": "shipment",
      "parcels": [
        {
          "contents": [
            {
              "customs_code": "610910",
              "description": "Synthetic cotton shirt",
              "quantity": 2,
              "unit_value": {
                "amount": "12.50",
                "currency": "XTS"
              }
            }
          ],
          "height_cm": 12,
          "length_cm": 30.5,
          "weight_grams": 1750,
          "width_cm": 20
        }
      ],
      "recipient": {
        "city": "Exampleton",
        "company": "Northstar Supplies",
        "country": "ZZ",
        "line1": "14 Fictional Quay",
        "line2": "Unit 7",
        "name": "Rowan Example",
        "phone": "+00000000000",
        "postal_code": "EX4 2PL",
        "region": "Test Province"
      },
      "reference": "synthetic-order-1042",
      "sender": {
        "city": "Exampleton",
        "company": "Northstar Supplies",
        "country": "ZZ",
        "line1": "14 Fictional Quay",
        "line2": "Unit 7",
        "name": "Rowan Example",
        "phone": "+00000000000",
        "postal_code": "EX4 2PL",
        "region": "Test Province"
      },
      "service": "economy",
      "status": "canceled",
      "tracking_number": "PE-000000000001",
      "updated_at": "2026-09-04T08:16:12Z"
    }
  ],
  "has_more": true,
  "next_cursor": "string"
}
```

Paginated (`cursor` scheme) — see [pagination.md](pagination.md#listshipments).

<a id="retrieveshipment"></a>
### GET /v1/shipments/{shipment_id}

Retrieve a shipment

`GET /v1/shipments/{shipment_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| shipment_id | path | string | yes |

Response `200`: `Shipment`.

_Example (synthesized from schema):_

```json
{
  "created_at": "2026-09-04T08:15:30Z",
  "delivery_instructions": "string",
  "id": "shp_00000000000000000001",
  "metadata": {
    "additionalProp1": "string"
  },
  "object": "shipment",
  "parcels": [
    {
      "contents": [
        {
          "customs_code": "610910",
          "description": "Synthetic cotton shirt",
          "quantity": 2,
          "unit_value": {
            "amount": "12.50",
            "currency": "XTS"
          }
        }
      ],
      "height_cm": 12,
      "length_cm": 30.5,
      "weight_grams": 1750,
      "width_cm": 20
    }
  ],
  "recipient": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "reference": "synthetic-order-1042",
  "sender": {
    "city": "Exampleton",
    "company": "Northstar Supplies",
    "country": "ZZ",
    "line1": "14 Fictional Quay",
    "line2": "Unit 7",
    "name": "Rowan Example",
    "phone": "+00000000000",
    "postal_code": "EX4 2PL",
    "region": "Test Province"
  },
  "service": "economy",
  "status": "canceled",
  "tracking_number": "PE-000000000001",
  "updated_at": "2026-09-04T08:16:12Z"
}
```
