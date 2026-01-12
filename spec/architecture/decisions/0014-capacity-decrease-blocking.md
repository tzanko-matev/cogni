# ADR 0014: Capacity Decrease via Blocking and Deferred Apply

## Status

- Proposed

## Context

- If capacity is decreased below current in-use reservations, immediate reduction is impossible without violating account constraints.
- We want predictable behavior that does not under-enforce limits.

## Decision

- When a limit capacity is decreased, the system enters a `decreasing` state for that key.
- While decreasing, **new reservations that include this key are denied** with a large `retry_after_ms` hint.
- The system waits until available capacity is at least the decrease delta, then applies the decrease and resumes accepting reservations.

## Specification

### Limit state

```go
type LimitState struct {
  Definition        LimitDefinition
  Status            string // "active" | "decreasing"
  PendingDecreaseTo uint64
}
```

### Decrease algorithm (shared semantics)

```go
func RequestDecrease(key LimitKey, newCap uint64) {
  state := registry[key]
  if newCap >= state.Definition.Capacity {
    applyIncreaseOrNoop(key, newCap)
    return
  }

  state.Status = "decreasing"
  state.PendingDecreaseTo = newCap
  registry[key] = state
}

func TryApplyDecrease(key LimitKey) {
  state := registry[key]
  if state.Status != "decreasing" {
    return
  }

  delta := state.Definition.Capacity - state.PendingDecreaseTo
  if availableCapacity(key) < delta {
    return
  }

  applyDecrease(key, delta)
  state.Definition.Capacity = state.PendingDecreaseTo
  state.PendingDecreaseTo = 0
  state.Status = "active"
  registry[key] = state
}
```

### Reserve behavior

- If any requirement key is `decreasing`, return `allowed=false` with `error=limit_decreasing:<key>`.
- Use a large `retry_after_ms` (configurable; e.g., 10000ms).

## Consequences

- Positive: Safe and deterministic; avoids violating account constraints.
- Negative: Temporarily denies new work until the decrease can be applied.

## Alternatives considered

- Immediate decrease (rejected: violates constraints).
- Best-effort one-time decrease (rejected: ambiguous, hard to reason about).
