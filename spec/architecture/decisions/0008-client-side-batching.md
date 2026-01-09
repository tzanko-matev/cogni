# ADR 0008: Client-Side Batching for Reserve/Complete

## Status

- Proposed

## Context

- Server-side microbatching reduces TigerBeetle overhead, but the HTTP server can still become a bottleneck if every job attempt uses its own HTTP request.
- The client already performs retries and scheduling; it is the best place to aggregate many small Reserve/Complete calls.
- Batching must preserve per-item semantics (LeaseID idempotency, retry rules, linked chains) and must not introduce cross-item atomicity.

## Decision

- Add explicit batch APIs for both Reserve and Complete:
  - HTTP: `/v1/reserve/batch` and `/v1/complete/batch`
  - In-process: `BatchReserve` and `BatchComplete` methods in the client library
- Each batch item is processed independently; results are returned in the same order as the request array.
- The client library includes a `Batcher` that aggregates calls up to a size or flush interval.

## API specification (HTTP)

### `POST /v1/reserve/batch`

Request:

```json
{
  "requests": [
    {
      "lease_id": "01J...",
      "job_id": "01J...",
      "requirements": [
        { "key": "global:llm:openai:gpt-4o:rpm", "amount": 1 }
      ]
    }
  ]
}
```

Rules:

- `requests` length must be `1..256` (server-enforced; configurable).
- Each item follows the same rules as single `Reserve`:
  - `lease_id` required, ULID
  - `requirements` length `1..32`
  - amount `>= 1`

Response (always HTTP 200 unless the whole batch is malformed):

```json
{
  "results": [
    {
      "allowed": true,
      "retry_after_ms": 0,
      "reserved_at_unix_ms": 1736660000000,
      "error": ""
    }
  ]
}
```

Notes:

- `error` is empty on success; on failure it contains a machine-parseable string (e.g., `unknown_limit_key`, `invalid_request`).
- `results[i]` corresponds to `requests[i]`.
- Denied reservations return `allowed=false` with a `retry_after_ms` hint.

### `POST /v1/complete/batch`

Request:

```json
{
  "requests": [
    {
      "lease_id": "01J...",
      "job_id": "01J...",
      "actuals": [
        { "key": "global:llm:openai:gpt-4o:tpm", "actual_amount": 740 }
      ]
    }
  ]
}
```

Rules:

- `requests` length must be `1..256`.
- Each item follows single `Complete` rules.

Response:

```json
{
  "results": [
    {
      "ok": true,
      "error": ""
    }
  ]
}
```

Notes:

- `results[i]` corresponds to `requests[i]`.
- If an item fails, `ok=false` and `error` is set. The batch itself still returns HTTP 200.

## In-process types (client library)

```go
// BatchReserveRequest mirrors the HTTP payload for in-process use.
type BatchReserveRequest struct {
  Requests []ReserveRequest
}

type BatchReserveResult struct {
  Allowed        bool
  RetryAfterMs   int
  ReservedAtUnix int64
  Error          string
}

type BatchReserveResponse struct {
  Results []BatchReserveResult
}

type BatchCompleteRequest struct {
  Requests []CompleteRequest
}

type BatchCompleteResult struct {
  Ok    bool
  Error string
}

type BatchCompleteResponse struct {
  Results []BatchCompleteResult
}

// Limiter supports both single and batch operations.
type Limiter interface {
  Reserve(ctx context.Context, req ReserveRequest) (ReserveResponse, error)
  Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
  BatchReserve(ctx context.Context, req BatchReserveRequest) (BatchReserveResponse, error)
  BatchComplete(ctx context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error)
}
```

## Client batcher algorithm (Go-like)

Key properties:

- Groups requests up to `maxBatch` or `flushInterval`.
- Preserves per-item semantics; no cross-item atomicity.
- Returns results to the caller with the same ordering.

```go
type batchItem struct {
  kind      string // "reserve" or "complete"
  req       any
  respCh    chan any
  errCh     chan error
}

func (b *Batcher) Run(ctx context.Context) {
  var pending []batchItem
  timer := time.NewTimer(b.flushInterval)
  defer timer.Stop()

  flush := func() {
    if len(pending) == 0 {
      return
    }
    items := pending
    pending = nil

    // Split by kind to avoid mixed payloads.
    reserves := make([]ReserveRequest, 0)
    completes := make([]CompleteRequest, 0)
    resIndex := make([]int, 0)
    compIndex := make([]int, 0)

    for i, item := range items {
      switch item.kind {
      case "reserve":
        reserves = append(reserves, item.req.(ReserveRequest))
        resIndex = append(resIndex, i)
      case "complete":
        completes = append(completes, item.req.(CompleteRequest))
        compIndex = append(compIndex, i)
      }
    }

    if len(reserves) > 0 {
      resp, err := b.limiter.BatchReserve(ctx, BatchReserveRequest{Requests: reserves})
      for i, idx := range resIndex {
        if err != nil {
          items[idx].errCh <- err
          continue
        }
        items[idx].respCh <- resp.Results[i]
      }
    }

    if len(completes) > 0 {
      resp, err := b.limiter.BatchComplete(ctx, BatchCompleteRequest{Requests: completes})
      for i, idx := range compIndex {
        if err != nil {
          items[idx].errCh <- err
          continue
        }
        items[idx].respCh <- resp.Results[i]
      }
    }
  }

  for {
    select {
    case <-ctx.Done():
      flush()
      return
    case item := <-b.in:
      pending = append(pending, item)
      if len(pending) >= b.maxBatch {
        flush()
        if !timer.Stop() {
          <-timer.C
        }
        timer.Reset(b.flushInterval)
      }
    case <-timer.C:
      flush()
      timer.Reset(b.flushInterval)
    }
  }
}
```

## Server-side handling (summary)

- Validate batch payload size and per-item request validity.
- For each item, call the existing Reserve/Complete path; do not enforce cross-item atomicity.
- Return per-item results in order.

## Consequences

- Positive: Reduces HTTP overhead and improves throughput under load; preserves per-item retry semantics.
- Negative: Adds API surface area and batching logic in the client.

## Alternatives considered

- Server-side batching only (rejected: too many HTTP requests from clients).
- Client-side batching without explicit API (rejected: would require per-item HTTP calls).
