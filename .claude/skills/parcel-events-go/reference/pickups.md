<!-- managed by Doctorine — edits will be overwritten -->
## Contents

- [DELETE /v1/pickups/{pickup_id}](#cancelpickup)
- [POST /v1/pickups](#createpickup)
- [GET /v1/pickups](#listpickups)
- [GET /v1/pickups/{pickup_id}](#retrievepickup)

# pickups

<a id="cancelpickup"></a>
### DELETE /v1/pickups/{pickup_id}

Cancel a pickup

`DELETE /v1/pickups/{pickup_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| pickup_id | path | string | yes |

Response `204`: no body.

<a id="createpickup"></a>
### POST /v1/pickups

Schedule a carrier pickup

`POST /v1/pickups`

| name | in | type | required |
| --- | --- | --- | --- |
| Idempotency-Key | header | string | yes |

Request body (`application/json`): `CreatePickupRequest` (required).

_Example (synthesized from schema):_

```json
{
  "address": {
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
  "notes": "string",
  "shipment_ids": [
    "string"
  ],
  "window": {
    "ends_at": "2026-09-06T12:00:00Z",
    "starts_at": "2026-09-06T09:00:00Z"
  }
}
```

Response `201`: `Pickup`.

_Example (synthesized from schema):_

```json
{
  "address": {
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
  "created_at": "2026-06-09T00:00:00Z",
  "id": "pku_00000000000000000001",
  "object": "pickup",
  "shipment_ids": [
    "string"
  ],
  "status": "canceled",
  "window": {
    "ends_at": "2026-09-06T12:00:00Z",
    "starts_at": "2026-09-06T09:00:00Z"
  }
}
```

<a id="listpickups"></a>
### GET /v1/pickups

List scheduled pickups

`GET /v1/pickups`

| name | in | type | required |
| --- | --- | --- | --- |
| cursor | query | string | no |
| limit | query | integer | no |
| scheduled_on | query | string | no |

Response `200`: `PickupPage`.

_Example (synthesized from schema):_

```json
{
  "data": [
    {
      "address": {
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
      "created_at": "2026-06-09T00:00:00Z",
      "id": "pku_00000000000000000001",
      "object": "pickup",
      "shipment_ids": [
        "string"
      ],
      "status": "canceled",
      "window": {
        "ends_at": "2026-09-06T12:00:00Z",
        "starts_at": "2026-09-06T09:00:00Z"
      }
    }
  ],
  "has_more": true,
  "next_cursor": "string"
}
```

Paginated (`cursor` scheme) — see [pagination.md](pagination.md#listpickups).

<a id="retrievepickup"></a>
### GET /v1/pickups/{pickup_id}

Retrieve a pickup

`GET /v1/pickups/{pickup_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| pickup_id | path | string | yes |

Response `200`: `Pickup`.

_Example (synthesized from schema):_

```json
{
  "address": {
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
  "created_at": "2026-06-09T00:00:00Z",
  "id": "pku_00000000000000000001",
  "object": "pickup",
  "shipment_ids": [
    "string"
  ],
  "status": "canceled",
  "window": {
    "ends_at": "2026-09-06T12:00:00Z",
    "starts_at": "2026-09-06T09:00:00Z"
  }
}
```
