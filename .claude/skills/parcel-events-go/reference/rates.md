<!-- managed by Doctorine — edits will be overwritten -->
# rates

<a id="quoterates"></a>
### POST /v1/rates/quote

Quote available shipping services

`POST /v1/rates/quote`

Request body (`application/json`): `RateQuoteRequest` (required).

_Example (synthesized from schema):_

```json
{
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
  }
}
```

Response `200`: `RatesQuoteResponse`.

_Example (synthesized from schema):_

```json
{
  "data": [
    {
      "amount": {
        "amount": "string",
        "currency": "string"
      },
      "carrier": "string",
      "estimated_delivery_date": "string",
      "expires_at": "string",
      "id": "string",
      "service": "string"
    }
  ]
}
```
