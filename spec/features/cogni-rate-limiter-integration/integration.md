# Integration Design (v1)

This document specifies **how Cogni wires the limiter into the runner** and how the two modes are constructed.

## New package: `internal/ratelimit`

Create a small integration package to centralize limiter construction and defaults.

### Public surface

```go
// BuildLimiter constructs a Limiter based on config.
func BuildLimiter(cfg spec.Config, repoRoot string) (ratelimiter.Limiter, error)

// ResolveTaskWorkers returns the worker count for a task.
func ResolveTaskWorkers(cfg spec.Config, task spec.TaskConfig) int

// MaxOutputTokens returns the max output tokens to use for reservations.
func MaxOutputTokens(cfg spec.Config, task spec.TaskConfig) uint64
```

### Mode handling

```
disabled: return ratelimiter.NoopLimiter
remote:   return httpclient.NewWithTimeout(baseURL, timeout)
embedded: return local.NewMemoryLimiterFromFile(limitsPath)
```

Notes:

- `limitsPath` is resolved relative to the repo root if not absolute.
- `remote` mode **does not** call admin endpoints in v1.
- `BuildLimiter` should return a wrapped limiter with batching if enabled.

### Batching wrapper

If `batch.size > 1`, wrap the limiter with:

```go
ratelimiter.NewBatcher(limiter, size, time.Duration(flushMs)*time.Millisecond)
```

### HTTP client timeout

Add a constructor to `pkg/ratelimiter/httpclient`:

```go
func NewWithTimeout(baseURL string, timeout time.Duration) *Client
```

This sets `http.Client{Timeout: timeout}` to avoid hung requests.

### No-op limiter

Add `ratelimiter.NoopLimiter` (in `pkg/ratelimiter`) to satisfy the interface when mode is disabled:

- `Reserve` returns `{Allowed: true}`.
- `Complete` returns `{Ok: true}`.
- Batch calls return allowed/ok for each item.

## Runner integration

### Build once per run

In `runner.Run`, create the limiter **once** and reuse it for all tasks:

```go
limiter, err := ratelimit.BuildLimiter(cfg, repoRoot)
```

Store it in `RunParams` or pass it explicitly into `runTask` / `runQuestionTask`.
Expose a `RunDependencies.LimiterFactory` for tests.

### Scheduler per task

For each task, create a scheduler with the task’s worker count:

```
workers := ResolveTaskWorkers(cfg, task.Task)
sched := ratelimiter.NewScheduler(limiter, workers)
defer sched.Shutdown(ctxWithTimeout)
```

Rationale: tasks execute sequentially today, so a per-task scheduler is simple and
avoids cross-task queue coupling. The shared limiter still enforces global limits.

### Prompt and token sizing

When building a `ratelimiter.Job`:

- `Provider` and `Model` come from the selected agent.
- `Prompt` is the exact question prompt (or task prompt).
- `MaxOutputTokens` comes from `MaxOutputTokens(cfg, task)`.

Actual tokens reported back to `Complete` are:

```
uint64(callResult.Metrics.Tokens)
```

Use this even if the call ends with an error (best-effort reconciliation).

## Error handling

- If `BuildLimiter` fails, the run fails with `runtime_error`.
- If scheduler shutdown times out, treat as `runtime_error`.
- Limiter errors during Reserve/Complete are handled by the scheduler retry loop.

## Logging in concurrent mode

Wrap verbose output in a concurrency-safe writer:

```go
type lockedWriter struct { mu sync.Mutex; w io.Writer }
```

Use this wrapper for `verboseWriter` and `verboseLogWriter` when workers > 1.
