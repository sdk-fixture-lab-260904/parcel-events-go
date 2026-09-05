<!-- managed by Doctorine — edits will be overwritten -->
# Errors

| Error class | Statuses | Shape | Action |
| --- | --- | --- | --- |
| `AuthenticationError` | 401 | - | reauthenticate |
| `ConflictError` | 409 | - | retry-with-backoff |
| `NotFoundError` | 404 | - | fix-request |
| `PaymentRequiredError` | 402 | - | fix-request |
| `RateLimitError` | 429 | - | retry-with-backoff |
| `ServerError` | 5XX | - | retry-with-backoff |
| `ValidationError` | 422 | `/type` == `https://parcel-events.example/problems/validation` | fix-request |

Retry policy: up to 3 retries. Delay = min(250ms × 2^attempt, 8000ms), capped at 60000ms of total elapsed time.

Eligible statuses: 408, 409, 429, 5XX. Connection errors (no response received) are retried the same way.

When a response carries a `Retry-After` header, honor it in place of the computed delay.

## Retry rules (MUST / MUST NOT)

- MUST honor a `Retry-After` header when present, even if it exceeds the computed backoff delay.
- MUST NOT retry any status outside the eligible set above.
- MUST treat a non-JSON error body as a transport error, capturing the status code and request-id — it is never retried by inspecting its shape.
- MUST NOT retry a non-idempotent POST after an ambiguous failure unless the request carried the `Idempotency-Key` idempotency header.
