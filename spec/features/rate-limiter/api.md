# Rate Limiter API Spec (v1)

All endpoints are HTTP+JSON. v1 has no authn/authz; deploy on trusted networks only.

## Common rules

- JSON keys are snake_case.
- `lease_id` is required and must be a ULID string.
- `job_id` is optional and used only for logging.
- Amounts are unsigned integers (`>= 1`).
- Unknown limit keys return `allowed=false` with an error string.
- For batch APIs, results are returned in the same order as requests.
- If the whole payload is malformed (missing fields, invalid JSON), respond with HTTP 400 and a JSON error.

### Error strings (machine-readable)

- `invalid_request`
- `unknown_limit_key:<key>`
- `backend_error`
- `limit_decreasing:<key>`

## Reserve (single)

### `POST /v1/reserve`

Request:

```json
{
  "lease_id": "01J...",
  "job_id": "01J...",
  "requirements": [
    { "key": "global:llm:openai:gpt-4o:rpm", "amount": 1 },
    { "key": "global:llm:openai:gpt-4o:tpm", "amount": 1800 },
    { "key": "global:llm:openai:gpt-4o:concurrency", "amount": 1 },
    { "key": "tenant:tenant_a:llm:daily_tokens", "amount": 1800 }
  ]
}
```

Validation:

- `requirements` length 1..32
- each key must exist in registry

Response (allowed):

```json
{
  "allowed": true,
  "retry_after_ms": 0,
  "reserved_at_unix_ms": 1736660000000,
  "error": ""
}
```

Response (denied):

```json
{
  "allowed": false,
  "retry_after_ms": 120,
  "reserved_at_unix_ms": 0,
  "error": ""
}
```

Response (unknown key):

```json
{
  "allowed": false,
  "retry_after_ms": 0,
  "reserved_at_unix_ms": 0,
  "error": "unknown_limit_key: tenant:foo:llm:daily_tokens"
}
```

Response (decreasing limit):

```json
{
  "allowed": false,
  "retry_after_ms": 10000,
  "reserved_at_unix_ms": 0,
  "error": "limit_decreasing: global:llm:openai:gpt-4o:tpm"
}
```

## Complete (single)

### `POST /v1/complete`

Request:

```json
{
  "lease_id": "01J...",
  "job_id": "01J...",
  "actuals": [
    { "key": "global:llm:openai:gpt-4o:tpm", "actual_amount": 740 },
    { "key": "tenant:tenant_a:llm:daily_tokens", "actual_amount": 740 }
  ]
}
```

Rules:

- `actuals` may be empty if actual usage is unknown.
- Concurrency is always released on Complete.

Response:

```json
{ "ok": true, "error": "" }
```

## Reserve (batch)

### `POST /v1/reserve/batch`

Request:

```json
{
  "requests": [
    { "lease_id": "01J...", "job_id": "01J...", "requirements": [ ... ] }
  ]
}
```

Rules:

- `requests` length 1..256 (server config).
- Each item follows the single Reserve validation rules.

Response:

```json
{
  "results": [
    { "allowed": true, "retry_after_ms": 0, "reserved_at_unix_ms": 1736660000000, "error": "" }
  ]
}
```

## Complete (batch)

### `POST /v1/complete/batch`

Request:

```json
{
  "requests": [
    { "lease_id": "01J...", "job_id": "01J...", "actuals": [ ... ] }
  ]
}
```

Rules:

- `requests` length 1..256 (server config).

Response:

```json
{
  "results": [
    { "ok": true, "error": "" }
  ]
}
```

## Admin API

### `PUT /v1/admin/limits`

Request:

```json
{
  "key": "global:llm:openai:gpt-4o:rpm",
  "kind": "rolling",
  "capacity": 3000,
  "window_seconds": 60,
  "timeout_seconds": 0,
  "unit": "requests",
  "description": "OpenAI gpt-4o requests per minute",
  "overage": "debt"
}
```

Response:

```json
{ "ok": true, "status": "active | decreasing" }
```

Rules:

- `kind` is `rolling` or `concurrency`.
- `capacity > 0`.
- `window_seconds > 0` for rolling only.
- `timeout_seconds > 0` for concurrency only.
- `overage` is `deny` or `debt` (default `debt`).
- If a decrease is attempted (new capacity < current capacity), the limit enters `decreasing` state and new reservations for that key are denied until the decrease is applied.

### `GET /v1/admin/limits`

Response:

```json
{ "limits": [ /* array of LimitInfo */ ] }
```

### `GET /v1/admin/limits/{key}`

Response (found):

```json
{ "limit": { /* LimitInfo */ } }
```

Response (not found): HTTP 404

### LimitInfo

```json
{
  "definition": { /* LimitDefinition */ },
  "status": "active | decreasing",
  "pending_decrease_to": 0
}
```

Notes:

- `pending_decrease_to` is set only when `status=decreasing`.
- While decreasing, Reserve requests that include the limit key return `allowed=false` with a large `retry_after_ms`.

## Idempotency rules (client)

- If Reserve times out or fails with transport error, retry with the SAME `lease_id`.
- If Reserve is denied, retry later with a NEW `lease_id`.
- Complete is best-effort; failures do not free capacity.

Next: `backend-tb.md`
