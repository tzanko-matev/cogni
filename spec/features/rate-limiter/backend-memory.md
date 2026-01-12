# In-Memory Backend Spec (v1)

The in-memory backend mirrors the TigerBeetle semantics using in-process data structures. It is used in single-binary mode.

## Data structures

```go
type rollingLimit struct {
  cap   uint64
  used  uint64
  heap  reservationHeap // min-heap by expiresAt
  byID  map[string]*reservation
}

type reservation struct {
  id        string
  amount    uint64
  expiresAt time.Time
  canceled  bool
  heapIndex int
}

type concLimit struct {
  cap   uint64
  holds map[string]time.Time // lease_id -> expiresAt
  heap  concHeap
}

type LimitState struct {
  Definition        LimitDefinition
  Status            string // "active" | "decreasing"
  PendingDecreaseTo uint64
}

type MemoryBackend struct {
  defs   map[LimitKey]LimitDefinition
  states map[LimitKey]LimitState
  roll   map[LimitKey]*rollingLimit
  conc   map[LimitKey]*concLimit
  debt   map[LimitKey]uint64
  leases map[string]LeaseState
  mu     sync.Mutex
}
```

## ApplyDefinition (admin)

```go
func (m *MemoryBackend) ApplyDefinition(def LimitDefinition) error {
  m.mu.Lock()
  defer m.mu.Unlock()

  prev, ok := m.defs[def.Key]
  if !ok {
    m.defs[def.Key] = def
    m.states[def.Key] = LimitState{Definition: def, Status: "active"}
    ensureLimitStores(def)
    return nil
  }

  if def.Capacity >= prev.Capacity {
    // increase or same
    m.defs[def.Key] = def
    m.states[def.Key] = LimitState{Definition: def, Status: "active"}
    updateCapacity(def)
    return nil
  }

  // decrease: mark as decreasing, block new reservations for this key
  m.states[def.Key] = LimitState{Definition: prev, Status: "decreasing", PendingDecreaseTo: def.Capacity}
  return nil
}
```

Decrease reconciliation (called periodically):

```go
func (m *MemoryBackend) TryApplyDecrease(key LimitKey) {
  state := m.states[key]
  if state.Status != "decreasing" {
    return
  }

  current := state.Definition.Capacity
  target := state.PendingDecreaseTo
  delta := current - target

  available := availableCapacity(key) // rolling: cap-used; concurrency: cap-len(holds)
  if available < delta {
    return
  }

  // apply decrease
  state.Definition.Capacity = target
  state.Status = "active"
  state.PendingDecreaseTo = 0
  m.states[key] = state
  m.defs[key] = state.Definition
  updateCapacity(state.Definition)
}
```

## Reserve algorithm

```go
func (m *MemoryBackend) Reserve(req ReserveRequest, now time.Time) ReserveResponse {
  m.mu.Lock()
  defer m.mu.Unlock()

  // deny if any limit is in decreasing state
  for _, r := range req.Requirements {
    if state := m.states[r.Key]; state.Status == "decreasing" {
      return ReserveResponse{Allowed: false, RetryAfterMs: decreaseRetryMs, Error: "limit_decreasing:" + string(r.Key)}
    }
  }

  // 1) validate definitions
  for _, r := range req.Requirements {
    if _, ok := m.defs[r.Key]; !ok {
      return ReserveResponse{Allowed: false, Error: "unknown_limit_key:" + string(r.Key)}
    }
  }

  // 2) cleanup expired reservations per key
  for _, r := range req.Requirements {
    def := m.defs[r.Key]
    if def.Kind == KindRolling {
      cleanupRolling(m.roll[r.Key], now)
    } else {
      cleanupConcurrency(m.conc[r.Key], now)
    }
  }

  // 3) check feasibility for all requirements
  for _, r := range req.Requirements {
    def := m.defs[r.Key]
    if def.Kind == KindRolling {
      if m.roll[r.Key].used+r.Amount > m.roll[r.Key].cap {
        return ReserveResponse{Allowed: false, RetryAfterMs: retryAfter(def)}
      }
    } else {
      if uint64(len(m.conc[r.Key].holds)+1) > m.conc[r.Key].cap {
        return ReserveResponse{Allowed: false, RetryAfterMs: retryAfter(def)}
      }
    }
  }

  // 4) apply all reservations
  for _, r := range req.Requirements {
    def := m.defs[r.Key]
    if def.Kind == KindRolling {
      addRollingReservation(m.roll[r.Key], req.LeaseID, r.Amount, now.Add(time.Duration(def.WindowSeconds)*time.Second))
    } else {
      addConcurrencyHold(m.conc[r.Key], req.LeaseID, now.Add(time.Duration(def.TimeoutSeconds)*time.Second))
    }
  }

  // 5) cache lease metadata
  m.leases[req.LeaseID] = LeaseState{
    LeaseID:         req.LeaseID,
    ReservedAtUnix:  now.UnixMilli(),
    Requirements:    req.Requirements,
    ReservedAmounts: indexByKey(req.Requirements),
  }

  return ReserveResponse{Allowed: true, ReservedAtUnixMs: now.UnixMilli()}
}
```

## Complete algorithm

```go
func (m *MemoryBackend) Complete(req CompleteRequest, now time.Time) CompleteResponse {
  m.mu.Lock()
  defer m.mu.Unlock()

  state, ok := m.leases[req.LeaseID]
  if !ok {
    return CompleteResponse{Ok: true}
  }

  // release concurrency holds
  for _, r := range state.Requirements {
    def := m.defs[r.Key]
    if def.Kind == KindConcurrency {
      delete(m.conc[r.Key].holds, req.LeaseID)
    }
  }

  // reconcile rolling
  for _, actual := range req.Actuals {
    def := m.defs[actual.Key]
    reserved := state.ReservedAmounts[actual.Key]
    if actual.ActualAmount < reserved {
      reduceRollingReservation(m.roll[actual.Key], req.LeaseID, actual.ActualAmount)
    } else if actual.ActualAmount > reserved && def.Overage == OverageDebt {
      m.debt[actual.Key] += actual.ActualAmount - reserved
    }
  }

  delete(m.leases, req.LeaseID)
  return CompleteResponse{Ok: true}
}
```

## Notes

- Capacity decrease: when a new capacity is lower than current, mark state as `decreasing` and deny new reservations that include the key until usage drops enough. Then apply the decrease and clear the state.
- `decreaseRetryMs` is a configured constant (see ADR 0010).
- Memory backend uses a single global mutex for correctness (v1).
- Cleanup of expired reservations is best-effort and performed on Reserve.
- Missing lease metadata skips reconciliation and leaves reservations to expire.
- Debt tracking is a simple counter per limit key.

Next: `client-lib.md`
