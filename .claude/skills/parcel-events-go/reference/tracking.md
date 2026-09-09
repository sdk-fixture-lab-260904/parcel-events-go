<!-- managed by Doctorine — edits will be overwritten -->
# tracking

<a id="retrievetrackingtimeline"></a>
### GET /v1/tracking/{tracking_number}

Retrieve a public tracking timeline

`GET /v1/tracking/{tracking_number}`

| name | in | type | required |
| --- | --- | --- | --- |
| tracking_number | path | string | yes |

Response `200`: `TrackingTimeline`.

_Example (synthesized from schema):_

```json
{
  "events": [
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
  "shipment": {
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
}
```
