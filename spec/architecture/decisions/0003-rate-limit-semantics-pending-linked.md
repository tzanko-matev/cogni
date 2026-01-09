# ADR 0003: Pending Transfers and Linked Chains for Rate Limiting Semantics

## Status

- Proposed

## Context

- LLM calls require multi-dimensional limits (RPM/TPM/concurrency/budgets) with atomic allow/deny.
- TigerBeetle supports pending transfers with timeouts and linked transfers for atomic chains.
- The in-memory backend must match TigerBeetle semantics.
- We will not persist reservation metadata across restarts in v1.

## Decision

- Rolling-window limits use pending transfers with a timeout equal to the window duration.
- Concurrency limits use pending transfers with a timeout; the reservation is voided on completion.
- Multi-resource requests use linked transfers so all requirements succeed or fail as a unit.
- The in-memory backend mirrors these semantics: rolling expiries restore capacity, and concurrency holds expire or are released on completion.
- Reservation metadata is stored in an in-memory lease cache only (no persistence). If cache data is missing, reconciliation is skipped and reservations expire naturally.

## Specification

### Lease cache (server-side, in-memory only)

```go
// LeaseState is stored only in memory, not persisted.
type LeaseState struct {
  LeaseID         string
  ReservedAtUnix  int64
  Requirements    []Requirement // includes concurrency keys
  ReservedAmounts map[LimitKey]uint64
}

type LeaseCache interface {
  Put(state LeaseState)
  Get(leaseID string) (LeaseState, bool)
  Delete(leaseID string)
}
```

### Reserve flow (TigerBeetle backend)

Given `ReserveRequest` with requirements `{key, amount}`:

```go
func Reserve(req ReserveRequest, defs map[LimitKey]LimitDefinition, now time.Time) ReserveResponse {
  transfers := []Transfer{}
  for i, r := range req.Requirements {
    def := defs[r.Key]
    timeout := 0
    if def.Kind == KindRolling {
      timeout = def.WindowSeconds
    } else if def.Kind == KindConcurrency {
      timeout = def.TimeoutSeconds
    }
    t := Transfer{
      ID:      transferID("reserve", req.LeaseID, r.Key),
      Debit:   limitAccountID(r.Key),
      Credit:  operatorAccountID(),
      Amount:  r.Amount,
      Pending: true,
      Timeout: timeout,
      Linked:  i < len(req.Requirements)-1,
    }
    transfers = append(transfers, t)
  }

  res := tb.CreateTransfers(transfers)
  if res.HasFailure("exceeds_credits") {
    return ReserveResponse{Allowed: false, RetryAfterMs: 0}
  }
  if res.HasFailure("unknown") {
    return ReserveResponse{Allowed: false, Error: "backend_error"}
  }

  // Store lease state for reconciliation (in-memory only).
  cache.Put(LeaseState{
    LeaseID:         req.LeaseID,
    ReservedAtUnix:  now.UnixMilli(),
    Requirements:    req.Requirements,
    ReservedAmounts: indexByKey(req.Requirements),
  })

  return ReserveResponse{Allowed: true, ReservedAtUnixMs: now.UnixMilli()}
}
```

Key points:

- All transfers are part of a single linked chain.
- If any transfer fails, the entire chain fails; no partial reservations exist.

### Complete flow (TigerBeetle backend)

```go
func Complete(req CompleteRequest, defs map[LimitKey]LimitDefinition, now time.Time) CompleteResponse {
  state, ok := cache.Get(req.LeaseID)
  if !ok {
    // No reconciliation info. Leave reservations to expire naturally.
    return CompleteResponse{Ok: true}
  }

  // 1) release concurrency holds
  for _, r := range state.Requirements {
    if defs[r.Key].Kind == KindConcurrency {
      tb.VoidPending(transferID("reserve", req.LeaseID, r.Key))
    }
  }

  // 2) reconcile rolling limits where actual < reserved
  for _, actual := range req.Actuals {
    reserved := state.ReservedAmounts[actual.Key]
    if actual.ActualAmount < reserved {
      tb.VoidPending(transferID("reserve", req.LeaseID, actual.Key))
      def := defs[actual.Key]
      remaining := max(1, def.WindowSeconds-int(elapsedSeconds(state.ReservedAtUnix, now)))
      tb.CreatePending(transferID("rereserve", req.LeaseID, actual.Key), amount=actual.ActualAmount, timeout=remaining)
    }
  }

  cache.Delete(req.LeaseID)
  return CompleteResponse{Ok: true}
}
```

Notes:

- If a pending transfer has already expired, `pending_transfer_expired` is treated as released.
- Reconciliation is best-effort and skipped if the lease cache is missing (e.g., after restart). This favors overestimation and protects capacity.

### In-memory backend semantics

Data structures:

```go
type rollingLimit struct {
  cap   uint64
  used  uint64
  heap  reservationHeap // min-heap by expiresAt
  byID  map[string]*reservation
}

type concLimit struct {
  cap   uint64
  holds map[string]time.Time // lease_id -> expiresAt
  heap  concHeap
}
```

Reserve algorithm (atomic):

```go
lock()
cleanupExpired(now)
if any requirement infeasible:
  return denied
apply all reservations
unlock()
```

Complete algorithm:

```go
lock()
release concurrency hold
for each actual:
  if actual < reserved:
    reduce reservation to actual
unlock()
```

## Consequences

- Positive: Atomic multi-limit enforcement; consistent behavior across backends.
- Negative: No reconciliation across server restarts; expired reservations may temporarily reduce throughput.

## Alternatives considered

- Persistent lease metadata (rejected for v1; complexity).
- Independent per-limit checks without atomicity (rejected: allows partial reservation).
