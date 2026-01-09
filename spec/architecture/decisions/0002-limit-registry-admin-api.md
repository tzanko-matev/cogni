# ADR 0002: Limit Registry and Admin API with Persistent Definitions

## Status

- Proposed

## Context

- Rate limit definitions must be adjustable at runtime without restarting TigerBeetle.
- The in-memory backend also needs the same limit definitions.
- Clients must fail fast on unknown limit keys.

## Decision

- Maintain a server-side registry of `LimitDefinition` entries keyed by `LimitKey`.
- Persist the registry to a JSON file with atomic rewrite (`limits.json.tmp` -> `limits.json`).
- Provide admin endpoints to create/update and read definitions:
  - `PUT /v1/admin/limits`
  - `GET /v1/admin/limits`
  - `GET /v1/admin/limits/{key}`
- On `Reserve`, reject unknown keys with a clear error.

## Specification

### Types

```go
// LimitKey identifies the resource being limited.
type LimitKey string

// LimitDefinition describes one limit.
type LimitDefinition struct {
  Key            LimitKey
  Kind           LimitKind
  Capacity       uint64
  WindowSeconds  int
  TimeoutSeconds int
  Unit           string
  Description    string
  Overage        OveragePolicy
}
```

### Registry data layout

- File path: `data/limits.json`
- JSON schema: array of `LimitDefinition` objects

Example:

```json
[
  {
    "key": "global:llm:openai:gpt-4o:rpm",
    "kind": "rolling",
    "capacity": 3000,
    "window_seconds": 60,
    "timeout_seconds": 0,
    "unit": "requests",
    "description": "OpenAI gpt-4o requests per minute",
    "overage": "deny"
  }
]
```

### Persistence algorithm (pseudo-code)

```go
func SaveRegistry(path string, defs []LimitDefinition) error {
  tmp := path + ".tmp"
  data := json.Marshal(defs)
  writeFile(tmp, data, 0644)
  fsync(tmp)
  rename(tmp, path)
  return nil
}
```

### Admin API

#### `PUT /v1/admin/limits`

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
  "overage": "deny"
}
```

Response:

```json
{ "ok": true }
```

Validation:

- `key` required
- `kind` in `{rolling, concurrency}`
- `capacity > 0`
- `window_seconds > 0` only for rolling
- `timeout_seconds > 0` only for concurrency
- `overage` in `{deny, debt}` (defaults to `deny`)

Side effects:

- Upsert definition in registry
- Persist `limits.json`
- Call backend `ApplyDefinition(def)`

#### `GET /v1/admin/limits`

Response:

```json
{ "limits": [ /* array of LimitDefinition */ ] }
```

#### `GET /v1/admin/limits/{key}`

Response (found):

```json
{ "limit": { /* LimitDefinition */ } }
```

Response (not found): HTTP 404

### Reserve validation

If any requirement key is missing in the registry:

```json
{
  "allowed": false,
  "retry_after_ms": 0,
  "reserved_at_unix_ms": 0,
  "error": "unknown_limit_key: <key>"
}
```

## Consequences

- Positive: Dynamic limits without restarts; shared definitions across backends.
- Negative: Requires config file permissions and atomic write handling.

## Alternatives considered

- Static limits baked into config only (rejected: requires redeploy/restart).
- TigerBeetle-only metadata (rejected: in-memory backend still needs definitions).
