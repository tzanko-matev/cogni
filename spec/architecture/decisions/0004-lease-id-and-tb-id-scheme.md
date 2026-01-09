# ADR 0004: LeaseID Attempt Semantics and Deterministic TigerBeetle IDs

## Status

- Proposed

## Context

- TigerBeetle transfer IDs are u128 and a failed transfer ID cannot be retried (`id_already_failed`).
- We need idempotency for timeouts and retry safety for denials.
- Accounts and transfers must be derived deterministically from domain identifiers.

## Decision

- Introduce two identifiers:
  - `JobID`: stable logical job identifier (optional but recommended).
  - `LeaseID`: unique per reserve attempt; MUST change after a denial.
- Use deterministic u128 IDs derived from stable labels (e.g., `acct:limit:<LimitKey>`, `xfer:reserve:<lease_id>:<key>`).
- Derive u128 IDs via SHA-256(label) and take the first 16 bytes (little-endian). If the result is 0 or max, flip a bit and recheck.

## Specification

### ID types

```go
type JobID string   // ULID or UUID

type LeaseID string // ULID; unique per attempt
```

### ID derivation

```go
func ID128(label string) u128 {
  digest := sha256(label)
  id := littleEndianU128(digest[0:16])
  if id == 0 || id == maxU128 {
    id = id ^ 1 // flip lowest bit
  }
  return id
}
```

### Account IDs

- `acct:operator`
- `acct:limit:<LimitKey>`
- `acct:debt:<LimitKey>` (for debt-enabled limits)

### Transfer IDs

- `xfer:reserve:<lease_id>:<limit_key>`
- `xfer:void:<lease_id>:<limit_key>`
- `xfer:rereserve:<lease_id>:<limit_key>`
- `xfer:debt:<lease_id>:<limit_key>`

### Retry rules

- If Reserve times out and the result is unknown, retry with the same `LeaseID` (idempotent).
- If Reserve is denied, retry later with a new `LeaseID`.
- Error handling and pessimistic accounting are defined in ADR 0012.

## Consequences

- Positive: Supports idempotent retries on timeouts and safe retries after denial.
- Negative: Requires LeaseID generation and bookkeeping in clients/scheduler.

## Alternatives considered

- JobID-only IDs (rejected: retries after denial would be blocked).
- Random u128 IDs per transfer (rejected: harder to audit and reason about).
