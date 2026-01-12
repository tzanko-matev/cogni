# TigerBeetle Backend Spec (v1)

This backend implements the rate limiting semantics using TigerBeetle pending transfers and linked chains.

## TigerBeetle constraints (must obey)

- No authentication: never expose TB directly to clients.
- A TB client has at most one in-flight request.
- Default max batch size is 8189 events (server-configurable).
- Account and transfer IDs are u128 and must not be 0 or max.
- `create_transfers` returns only failures; successes are implicit.
- A failed transfer ID cannot be retried (`id_already_failed`).

## Account model

One ledger for all limits:

- `LEDGER_LIMITS = 1`
- `CODE_LIMIT = 1` for reserve transfers

Accounts:

- Operator account: `acct:operator`
- Resource account per limit: `acct:limit:<LimitKey>` with `debits_must_not_exceed_credits`
- Debt account per limit (when `overage=debt`): `acct:debt:<LimitKey>` without `debits_must_not_exceed_credits`

## Limit state (admin)

```go
type LimitState struct {
  Definition        LimitDefinition
  Status            string // "active" | "decreasing"
  PendingDecreaseTo uint64 // only when decreasing
}
```

LimitState is persisted in the registry file and loaded at startup.

## ID scheme

Use deterministic u128 IDs derived from SHA-256(label) first 16 bytes (little-endian). If ID is 0 or max, flip one bit.

Transfer IDs:

- `xfer:reserve:<lease_id>:<limit_key>`
- `xfer:void:<lease_id>:<limit_key>`
- `xfer:rereserve:<lease_id>:<limit_key>`
- `xfer:debt:<lease_id>:<limit_key>`

## ApplyDefinition (admin provisioning)

### Create accounts if missing

- Operator account: create once.
- Resource account: create if missing with `debits_must_not_exceed_credits`.
- Debt account: create if limit has `overage=debt`.

### Capacity changes

Both increases and decreases are supported.

Increase algorithm:

1) Read account balance for `acct:limit:<key>`.
2) If `balance < capacity`, post transfer from operator -> resource for `(capacity - balance)`.
3) If `balance >= capacity`, do nothing.

Decrease algorithm (blocking):

1) If `new_capacity < current_capacity`, set `LimitState.Status = "decreasing"` and `PendingDecreaseTo = new_capacity`.
2) While `status=decreasing`, **deny new reservations** that include this key.
3) Periodically check the resource account:
   - `balance = credits_posted - debits_posted`
   - `available = balance - debits_pending`
   - `delta = current_capacity - PendingDecreaseTo`
4) When `available >= delta`, post transfer `resource -> operator` for `delta`.
5) Update `current_capacity = PendingDecreaseTo`, clear `Status` and `PendingDecreaseTo`, and resume reservations.

Retry hints while decreasing:

- Reserve returns `allowed=false`, `error=limit_decreasing:<key>`.
- `retry_after_ms` uses a large configured value (e.g., 10000ms).

## Reserve flow

For each requirement, create a pending transfer resource -> operator with timeout:

- Rolling: `timeout = window_seconds`.
- Concurrency: `timeout = timeout_seconds`.

All transfers are linked in one chain.

Pseudo-code:

```go
func Reserve(req ReserveRequest, defs map[LimitKey]LimitDefinition, now time.Time) ReserveResponse {
  if anyRequirementIsDecreasing(req.Requirements) {
    return ReserveResponse{Allowed: false, RetryAfterMs: decreaseRetryMs, Error: "limit_decreasing:" + key}
  }
  transfers := buildPendingLinkedTransfers(req, defs)
  res := tb.CreateTransfers(transfers)
  if res.HasFailure("exceeds_credits") {
    return ReserveResponse{Allowed: false, RetryAfterMs: retryAfter(req, defs)}
  }
  if res.HasAnyFailure() {
    return ReserveResponse{Allowed: false, Error: "backend_error"}
  }

  cache.Put(LeaseState{
    LeaseID:         req.LeaseID,
    ReservedAtUnix:  now.UnixMilli(),
    Requirements:    req.Requirements,
    ReservedAmounts: indexByKey(req.Requirements),
  })
  return ReserveResponse{Allowed: true, ReservedAtUnixMs: now.UnixMilli()}
}
```

## Complete flow

### 1) Release concurrency holds

Void the original pending transfers for concurrency keys. If already expired, ignore.

### 2) Reconcile rolling limits (actual < reserved)

- Void the original pending transfer for that key.
- Re-reserve only `actual_amount` for the remaining window.

### 3) Overage (actual > reserved)

- Attempt to reserve the diff for the remaining window.
- If it fails and `overage=debt`, record a posted transfer from debt account -> operator.

Pseudo-code:

```go
func Complete(req CompleteRequest, defs map[LimitKey]LimitDefinition, now time.Time) CompleteResponse {
  state, ok := cache.Get(req.LeaseID)
  if !ok {
    return CompleteResponse{Ok: true} // best-effort, no reconciliation
  }

  releaseConcurrency(state)

  for _, actual := range req.Actuals {
    reserved := state.ReservedAmounts[actual.Key]
    def := defs[actual.Key]
    remaining := max(1, def.WindowSeconds-int(elapsedSeconds(state.ReservedAtUnix, now)))

    if actual.ActualAmount < reserved {
      voidReserve(req.LeaseID, actual.Key)
      reserveAmount(req.LeaseID, actual.Key, actual.ActualAmount, remaining)
    } else if actual.ActualAmount > reserved {
      diff := actual.ActualAmount - reserved
      if !reserveAmount(req.LeaseID, actual.Key, diff, remaining) && def.Overage == OverageDebt {
        recordDebt(req.LeaseID, actual.Key, diff)
      }
    }
  }

  cache.Delete(req.LeaseID)
  return CompleteResponse{Ok: true}
}
```

## Microbatching (server-side)

- See ADR 0005 for submitter design.
- The submitter must never split a linked chain across batches.
- Map failure indices back to their WorkItem.

## Retry-after heuristics

- Use ADR 0010 heuristics (concurrency vs rolling backoff).
- Use the max retry delay across failed requirements in a request.

## Lease metadata

- Stored in memory only (no persistence).
- Missing metadata means reconciliation is skipped; reservations expire naturally.

Next: `backend-memory.md`
