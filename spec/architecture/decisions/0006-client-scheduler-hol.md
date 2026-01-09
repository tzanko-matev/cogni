# ADR 0006: Client-Side Scheduler to Avoid Head-of-Line Blocking

## Status

- Proposed

## Context

- A single FIFO queue can block fast providers behind slow or rate-limited providers.
- Cogni may issue concurrent requests to multiple providers/models.
- The server should remain simple; queueing policy can live in the client.

## Decision

- Provide an optional client-side Scheduler that:
  - Maintains a queue per (provider, model).
  - Round-robins across ready queues.
  - On deny, sets a not-before timestamp and retries later with a new LeaseID.

## Specification

### Types

```go
type Job struct {
  JobID   string
  LeaseID string // set per attempt; regenerated on retries

  TenantID, Provider, Model string
  Prompt                    string
  MaxOutputTokens           uint64
  WantDailyBudget           bool

  Execute func(ctx context.Context) (actualTokens uint64, err error)
}

type Scheduler struct {
  limiter Limiter
  workers int
  queues  map[string]*workQueue // key = provider:model
}

func NewScheduler(l Limiter, workers int) *Scheduler
func (s *Scheduler) Submit(job Job)
func (s *Scheduler) Shutdown(ctx context.Context) error
```

### Algorithm (pseudo-code)

```go
for each worker:
  loop:
    q := nextReadyQueueRoundRobin()
    if q == nil:
      sleep(short)
      continue

    job := q.popReady()
    reqs := BuildLLMRequirements(job)

    res := limiter.Reserve(reqs)
    if res.Allowed {
      actual, err := job.Execute(ctx)
      limiter.Complete(lease_id, actual)
      continue
    }

    job.LeaseID = newLeaseID()
    job.notBefore = now + res.RetryAfterMs + jitter
    q.pushBlocked(job)
```

### Key behaviors

- Uses per-(provider,model) queues to avoid head-of-line blocking.
- Respects `retry_after_ms` hints when a reservation is denied.
- Regenerates `LeaseID` after denial to avoid `id_already_failed` on TB.

## Consequences

- Positive: Avoids head-of-line blocking without centralizing queueing in the server.
- Negative: Adds client complexity and requires careful LeaseID regeneration.

## Alternatives considered

- Single FIFO client queue (rejected: head-of-line blocking).
- Server-side global queue (rejected: adds central scheduling complexity).
