<!-- managed by Doctorine — edits will be overwritten -->
## Contents

- [POST /v1/webhook-endpoints](#createwebhookendpoint)
- [GET /v1/webhook-endpoints](#listwebhookendpoints)
- [DELETE /v1/webhook-endpoints/{endpoint_id}](#deletewebhookendpoint)
- [PATCH /v1/webhook-endpoints/{endpoint_id}](#updatewebhookendpoint)

# webhook endpoints

<a id="createwebhookendpoint"></a>
### POST /v1/webhook-endpoints

Register a webhook endpoint

`POST /v1/webhook-endpoints`

Request body (`application/json`): `CreateWebhookEndpointRequest` (required).

_Example (synthesized from schema):_

```json
{
  "description": "string",
  "events": [
    "pickup.completed"
  ],
  "url": "https://hooks.parcel-events.example/events"
}
```

Response `201`: `WebhookEndpoint`.

_Example (synthesized from schema):_

```json
{
  "created_at": "2026-06-09T00:00:00Z",
  "enabled": true,
  "events": [
    "pickup.completed"
  ],
  "id": "whe_00000000000000000001",
  "object": "webhook_endpoint",
  "signing_secret_hint": "synthetic-key-ending-0000",
  "url": "https://hooks.parcel-events.example/events"
}
```

<a id="listwebhookendpoints"></a>
### GET /v1/webhook-endpoints

List webhook endpoints

`GET /v1/webhook-endpoints`

| name | in | type | required |
| --- | --- | --- | --- |
| cursor | query | string | no |
| limit | query | integer | no |

Response `200`: `WebhookEndpointPage`.

_Example (synthesized from schema):_

```json
{
  "data": [
    {
      "created_at": "2026-06-09T00:00:00Z",
      "enabled": true,
      "events": [
        "pickup.completed"
      ],
      "id": "whe_00000000000000000001",
      "object": "webhook_endpoint",
      "signing_secret_hint": "synthetic-key-ending-0000",
      "url": "https://hooks.parcel-events.example/events"
    }
  ],
  "has_more": true,
  "next_cursor": "string"
}
```

Paginated (`cursor` scheme) — see [pagination.md](pagination.md#listwebhookendpoints).

<a id="deletewebhookendpoint"></a>
### DELETE /v1/webhook-endpoints/{endpoint_id}

Delete a webhook endpoint

`DELETE /v1/webhook-endpoints/{endpoint_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| endpoint_id | path | string | yes |

Response `204`: no body.

<a id="updatewebhookendpoint"></a>
### PATCH /v1/webhook-endpoints/{endpoint_id}

Update a webhook endpoint

`PATCH /v1/webhook-endpoints/{endpoint_id}`

| name | in | type | required |
| --- | --- | --- | --- |
| endpoint_id | path | string | yes |

Request body (`application/merge-patch+json`): `UpdateWebhookEndpointRequest` (required).

_Example (synthesized from schema):_

```json
{
  "enabled": true,
  "events": [
    "pickup.completed"
  ],
  "url": "https://example.com"
}
```

Response `200`: `WebhookEndpoint`.

_Example (synthesized from schema):_

```json
{
  "created_at": "2026-06-09T00:00:00Z",
  "enabled": true,
  "events": [
    "pickup.completed"
  ],
  "id": "whe_00000000000000000001",
  "object": "webhook_endpoint",
  "signing_secret_hint": "synthetic-key-ending-0000",
  "url": "https://hooks.parcel-events.example/events"
}
```
