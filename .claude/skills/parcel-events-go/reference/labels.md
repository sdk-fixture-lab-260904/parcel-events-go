<!-- managed by Doctorine — edits will be overwritten -->
# labels

<a id="createlabel"></a>
### POST /v1/labels

Buy and render a shipping label

`POST /v1/labels`

| name | in | type | required |
| --- | --- | --- | --- |
| Idempotency-Key | header | string | yes |

Request body (`application/json`): `CreateLabelRequest` (required).

_Example (synthesized from schema):_

```json
{
  "format": "pdf",
  "page_size": "four_by_six",
  "shipment_id": "shp_00000000000000000001"
}
```

Response `201`: `Label`.

_Example (synthesized from schema):_

```json
{
  "download_url": "https://downloads.parcel-events.example/labels/synthetic-label.pdf",
  "expires_at": "2026-06-09T00:00:00Z",
  "format": "pdf",
  "id": "lbl_00000000000000000001",
  "object": "label",
  "price": {
    "amount": "12.50",
    "currency": "XTS"
  },
  "shipment_id": "shp_00000000000000000001"
}
```

<a id="retrievelabel"></a>
### GET /v1/labels/{label_id}

Retrieve label metadata and download URL

`GET /v1/labels/{label_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| label_id | path | string | yes |

Response `200`: `Label`.

_Example (synthesized from schema):_

```json
{
  "download_url": "https://downloads.parcel-events.example/labels/synthetic-label.pdf",
  "expires_at": "2026-06-09T00:00:00Z",
  "format": "pdf",
  "id": "lbl_00000000000000000001",
  "object": "label",
  "price": {
    "amount": "12.50",
    "currency": "XTS"
  },
  "shipment_id": "shp_00000000000000000001"
}
```
