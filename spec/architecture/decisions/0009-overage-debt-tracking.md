# ADR 0009: Overage Handling via Debt Tracking

## Status

- Proposed

## Context

- LLM token usage is estimated before the call and reconciled afterward.
- Actual usage can exceed the reserved estimate.
- We do not want post-call reconciliation to fail or crash the pipeline; we want to track overage.

## Decision

- Use a debt account to record overages when actual usage exceeds the reservation and cannot be charged against the resource account.
- Debt tracking is enabled per limit via `LimitDefinition.Overage = "debt"`.

## Specification

### Account model (TigerBeetle)

For each debt-enabled limit key:

- Resource account: `acct:limit:<LimitKey>` with `debits_must_not_exceed_credits`
- Operator account: `acct:operator`
- Debt account: `acct:debt:<LimitKey>` without `debits_must_not_exceed_credits`

### Settlement algorithm (rolling limits only)

```go
func ReconcileOverage(leaseID string, key LimitKey, reserved, actual uint64, def LimitDefinition, reservedAt time.Time, now time.Time) {
  if actual <= reserved {
    return
  }
  diff := actual - reserved

  // Try to reserve the extra usage for the remainder of the window.
  remaining := max(1, def.WindowSeconds-int(elapsedSeconds(reservedAt, now)))
  err := tb.CreatePending(
    transferID("rereserve", leaseID, key),
    debit=limitAccountID(key),
    credit=operatorAccountID(),
    amount=diff,
    timeout=remaining,
  )
  if err == nil {
    return
  }

  // If we cannot reserve more, record debt (posted transfer).
  if def.Overage == OverageDebt {
    tb.CreatePosted(
      transferID("debt", leaseID, key),
      debit=debtAccountID(key),
      credit=operatorAccountID(),
      amount=diff,
    )
  }
}
```

### In-memory backend

- For debt-enabled limits, maintain a `debtByKey` counter (uint64) per limit key.
- On overage, increment `debtByKey[key] += diff`.
- Debt does not affect rate limiting capacity for the rolling window (it is purely accounting).

### Notes

- Debt applies only to rolling limits where `actual > reserved`.
- Concurrency limits never use debt tracking.
- Debt is a record of overage for reporting/billing; it does not free capacity.

## Consequences

- Positive: Overages are tracked without failing reconciliation; avoids undercounting usage.
- Negative: Some limits become "soft" on overage; debt handling requires downstream reporting/billing.

## Alternatives considered

- Deny/reject on overage (rejected: request already executed).
- Silent undercount (rejected: violates accuracy and billing).
