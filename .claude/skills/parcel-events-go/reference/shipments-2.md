<!-- managed by Doctorine — edits will be overwritten -->
# shipments (part 2)

<a id="updateshipment"></a>
### PATCH /v1/shipments/{shipment_id}

Update delivery instructions

`PATCH /v1/shipments/{shipment_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| shipment_id | path | string | yes |

Request body (`application/merge-patch+json`): `UpdateShipmentRequest` (required).

_Example (synthesized from schema):_

```json
{
  "delivery_instructions": "string",
  "metadata": {
    "additionalProp1": "string"
  }
}
```

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
