# ADR 0001: Pluggable Rate Limiter Backends and Deployment Modes

## Status

- Proposed

## Context

- Cogni needs rate limiting for LLM calls with both a SaaS/server mode and a single-binary local mode.
- TigerBeetle has no authentication, so it must not be exposed directly to clients.
- The design calls for a common API while swapping storage/ledger backends.

## Decision

- Implement a common rate limiter backend interface with two concrete backends:
  - Distributed mode: `ratelimiterd` server calling TigerBeetle.
  - Single-binary mode: in-memory backend used in-process without TigerBeetle.
- Clients call `ratelimiterd` in distributed mode; they never call TigerBeetle directly.

## Specification

### Backend interface (in-process)

```go
// LimitKey identifies the resource being limited.
type LimitKey string

// LimitKind defines the limiter semantics.
type LimitKind string

const (
  KindRolling     LimitKind = "rolling"
  KindConcurrency LimitKind = "concurrency"
)

// OveragePolicy defines what happens when actual usage exceeds the reservation.
type OveragePolicy string

const (
  OverageDeny OveragePolicy = "deny" // default
  OverageDebt OveragePolicy = "debt" // record overage
)

// LimitDefinition is the server-side definition for a limit.
type LimitDefinition struct {
  Key            LimitKey
  Kind           LimitKind
  Capacity       uint64
  WindowSeconds  int // rolling only
  TimeoutSeconds int // concurrency only
  Unit           string
  Description    string
  Overage        OveragePolicy
}

// Requirement is a requested reservation for a limit.
type Requirement struct {
  Key    LimitKey
  Amount uint64
}

// Actual reports the actual usage for reconciliation.
type Actual struct {
  Key          LimitKey
  ActualAmount uint64
}

// ReserveRequest/Response are shared between in-process and HTTP.
type ReserveRequest struct {
  LeaseID      string
  JobID        string
  Requirements []Requirement
}

type ReserveResponse struct {
  Allowed          bool
  RetryAfterMs     int
  ReservedAtUnixMs int64
  Error            string
}

type CompleteRequest struct {
  LeaseID string
  JobID   string
  Actuals []Actual
}

type CompleteResponse struct {
  Ok    bool
  Error string
}

// Backend supports both Reserve and Complete, and limit updates.
type Backend interface {
  ApplyDefinition(ctx context.Context, def LimitDefinition) error
  Reserve(ctx context.Context, req ReserveRequest, reservedAt time.Time) (ReserveResponse, error)
  Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
}
```

### Deployment modes

- **Distributed:** client → HTTP → `ratelimiterd` → TB backend
- **Single-binary:** client code calls memory backend directly

### Client interface

```go
// Limiter is the client-facing API. In distributed mode it wraps HTTP; in local mode it wraps memory.
type Limiter interface {
  Reserve(ctx context.Context, req ReserveRequest) (ReserveResponse, error)
  Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
}
```

## Consequences

- Positive: Enables both SaaS and local CLI usage with one codebase; avoids exposing TigerBeetle.
- Negative: Requires dual backend implementations and an HTTP server in distributed mode.

## Alternatives considered

- TigerBeetle-only deployment (rejected: no auth, no local mode).
- Memory-only rate limiter (rejected: no distributed scalability).
