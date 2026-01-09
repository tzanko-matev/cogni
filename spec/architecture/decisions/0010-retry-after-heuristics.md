# ADR 0010: Retry-After Hints Without Read Path

## Status

- Proposed

## Context

- The TigerBeetle hot path is write-only to keep latency low.
- Without reads, the server cannot compute exact time-to-availability.
- Clients need a `retry_after_ms` hint to avoid tight retry loops.

## Decision

- Compute `retry_after_ms` using configurable heuristics and backoff.
- Use different heuristics for concurrency vs rolling limits.

## Specification

### Configuration

```yaml
retry_policy:
  concurrency:
    base_ms: 50
    max_ms: 2000
    factor: 2.0
    jitter_ms: 25
  rolling:
    base_ms: 100
    max_ms: 5000
    factor: 1.5
    jitter_ms: 50
    window_fraction: 0.1
```

### Deny counters (server-side)

```go
// DenyTracker tracks recent denies by limit key.
type DenyTracker interface {
  Increment(key LimitKey, now time.Time) int // returns current streak
  Decay(key LimitKey, now time.Time) int     // called on allow
}
```

### Algorithm

```go
func RetryAfterMs(def LimitDefinition, streak int, cfg RetryPolicy) int {
  switch def.Kind {
  case KindConcurrency:
    // Backoff based on denial streak, capped by timeout and max.
    raw := float64(cfg.BaseMs) * math.Pow(cfg.Factor, float64(streak))
    capMs := minInt(cfg.MaxMs, def.TimeoutSeconds*1000)
    return withJitter(clampInt(int(raw), cfg.BaseMs, capMs), cfg.JitterMs)

  case KindRolling:
    // Base derived from window size; then apply backoff.
    base := maxInt(cfg.BaseMs, int(float64(def.WindowSeconds*1000)*cfg.WindowFraction))
    raw := float64(base) * math.Pow(cfg.Factor, float64(streak))
    return withJitter(clampInt(int(raw), base, cfg.MaxMs), cfg.JitterMs)

  default:
    return cfg.BaseMs
  }
}
```

### Usage

- On a denied reservation, use the max `retry_after_ms` across all failed requirements in that request.
- `streak` is maintained per `LimitKey` in a best-effort tracker; it is not durable and resets on restart.

## Consequences

- Positive: Provides bounded backoff without reads; reduces client retry churn.
- Negative: `retry_after_ms` is only a hint; it may be too conservative or optimistic.

## Alternatives considered

- Add read path for exact availability (rejected: increases latency and load).
- Fixed retry delay for all denies (rejected: poor adaptability under load).
