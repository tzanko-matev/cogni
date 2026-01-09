# ADR 0012: Error Model and Pessimistic Accounting

## Status

- Proposed

## Context

- Reserve/Complete are distributed calls that can timeout or fail.
- We prefer to overestimate usage rather than underestimate it.
- LeaseID semantics provide idempotency for retries.

## Decision

- Treat unknown outcomes pessimistically: do not proceed with an LLM call unless a reservation is confirmed allowed.
- On any uncertainty or error during Complete, do not free capacity; allow reservations to expire naturally.
- Use idempotent retries with the same LeaseID on timeouts or transport errors.

## Specification

### Error categories

- **Denied:** Reserve returns `allowed=false` (definitive). Client must generate a new LeaseID on retry.
- **Allowed:** Reserve returns `allowed=true` (definitive). Client may proceed with the LLM call.
- **Unknown:** Reserve request times out or transport errors occur. Client must retry with the same LeaseID until a definitive response is received or the caller aborts.

### Client rules

```go
func ReserveWithRetry(ctx context.Context, lim Limiter, req ReserveRequest) (ReserveResponse, error) {
  for {
    res, err := lim.Reserve(ctx, req)
    if err == nil {
      return res, nil
    }
    if ctx.Done() {
      return ReserveResponse{}, ctx.Err()
    }
    // Unknown outcome: retry with same LeaseID.
    sleep(backoff)
  }
}

func CompleteBestEffort(ctx context.Context, lim Limiter, req CompleteRequest) {
  // Best effort. If it fails, do not attempt to free capacity.
  _, _ = lim.Complete(ctx, req)
}
```

### Pessimistic reconciliation

- If `Complete` fails, skip reconciliation and allow pending reservations to expire naturally.
- This overestimates usage but avoids undercounting.

### Batch errors

- Batch endpoints return per-item results.
- Items with error responses are treated as **unknown** and retried with the same LeaseID.

## Consequences

- Positive: Prevents undercounting and accidental limit bypass.
- Negative: Potentially lower throughput under persistent network errors.

## Alternatives considered

- Optimistic retry with new LeaseID (rejected: risks double reservation).
- Freeing capacity on Complete failure (rejected: underestimates usage).
