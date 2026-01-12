# Client Library Spec (v1)

The client library provides:

- `Limiter` interface for Reserve/Complete.
- HTTP client for distributed mode.
- Local client for in-memory backend.
- Optional Scheduler to avoid head-of-line blocking.
- Optional Batcher to reduce HTTP overhead.

## Types

```go
// Public types shared with the API.
type LimitKey string

type Requirement struct {
  Key    LimitKey
  Amount uint64
}

type Actual struct {
  Key          LimitKey
  ActualAmount uint64
}

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

type Limiter interface {
  Reserve(ctx context.Context, req ReserveRequest) (ReserveResponse, error)
  Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
  BatchReserve(ctx context.Context, req BatchReserveRequest) (BatchReserveResponse, error)
  BatchComplete(ctx context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error)
}
```

## LLM requirement helper

```go
type LLMReserveInput struct {
  LeaseID         string
  JobID           string
  TenantID        string
  Provider        string
  Model           string
  Prompt          string
  MaxOutputTokens uint64
  WantDailyBudget bool
}

func EstimatePromptTokens(prompt string) uint64 {
  return uint64(len([]byte(prompt)))
}

func BuildLLMRequirements(in LLMReserveInput) []Requirement {
  upper := EstimatePromptTokens(in.Prompt) + in.MaxOutputTokens
  reqs := []Requirement{
    {Key: LimitKey(fmt.Sprintf("global:llm:%s:%s:rpm", in.Provider, in.Model)), Amount: 1},
    {Key: LimitKey(fmt.Sprintf("global:llm:%s:%s:tpm", in.Provider, in.Model)), Amount: upper},
    {Key: LimitKey(fmt.Sprintf("global:llm:%s:%s:concurrency", in.Provider, in.Model)), Amount: 1},
  }
  if in.WantDailyBudget {
    reqs = append(reqs, Requirement{
      Key: LimitKey(fmt.Sprintf("tenant:%s:llm:daily_tokens", in.TenantID)),
      Amount: upper,
    })
  }
  return reqs
}
```

## Scheduler (avoid head-of-line blocking)

```go
type Job struct {
  JobID   string
  LeaseID string // set per attempt

  TenantID, Provider, Model string
  Prompt                    string
  MaxOutputTokens           uint64
  WantDailyBudget           bool

  Execute func(ctx context.Context) (actualTokens uint64, err error)
}

type Scheduler struct { /* internal queues */ }

func NewScheduler(l Limiter, workers int) *Scheduler
func (s *Scheduler) Submit(job Job)
func (s *Scheduler) Shutdown(ctx context.Context) error
```

Algorithm highlights:

- Maintain a queue per `(provider,model)`.
- Round-robin across ready queues.
- On denial, use `retry_after_ms` + jitter and regenerate LeaseID.
- On unknown Reserve outcome, retry with the same LeaseID.

## Batcher (client-side batching)

```go
type Batcher struct {
  maxBatch      int
  flushInterval time.Duration
  limiter       Limiter
}

func NewBatcher(l Limiter, maxBatch int, flushInterval time.Duration) *Batcher
```

Behavior:

- Aggregate Reserve and Complete requests into batch payloads.
- Preserve per-item ordering and semantics.
- Do not mix Reserve and Complete in the same batch call.

## Error handling rules

- Reserve timeout/transport error => retry with same LeaseID.
- Reserve denied => retry later with new LeaseID.
- Reserve denied with `limit_decreasing:<key>` => retry after `retry_after_ms` (large).
- Complete errors are best-effort; do not attempt to free capacity on failure.

## Usage examples

### Distributed mode

```go
c := httpclient.New("http://ratelimiter:8080")
lim := ratelimiter.NewBatcher(c, 128, 2*time.Millisecond)

sched := ratelimiter.NewScheduler(lim, 32)

sched.Submit(Job{ /* ... */ })
```

### Single-binary mode

```go
lim := local.NewMemoryLimiterFromFile("./limits.json")
lim = ratelimiter.NewBatcher(lim, 128, 2*time.Millisecond)

sched := ratelimiter.NewScheduler(lim, 8)
```

Next: `implementation-plan.md`
