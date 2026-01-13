# Cogni Rate Limiter Integration Plan (v1)

Each step must include tests with explicit timeouts. Use `.feature` files when behavior is user-visible.

## Step 1: Config schema + validation

- Add `RateLimiterConfig` to `spec.Config`.
- Add `TaskConfig.Concurrency`.
- Update normalization defaults.
- Update validation rules (see `config.md`).

Tests:
- `internal/config/config_rate_limiter_test.go` (timeout <= 1s per test).

## Step 2: Limiter construction helpers

- Add `internal/ratelimit` package with `BuildLimiter`, `ResolveTaskWorkers`, `MaxOutputTokens`.
- Add `ratelimiter.NoopLimiter` in `pkg/ratelimiter`.
- Add `httpclient.NewWithTimeout`.
- Add batching wrapper logic.

Tests:
- `internal/ratelimit/limiter_test.go` (timeout <= 1s).
- `pkg/ratelimiter/httpclient` tests for timeout constructor (timeout <= 1s).

## Step 3: Runner wiring (rate limiting for all calls)

- In `runner.Run`, build the limiter once and pass it into task runners.
- Add dependency injection for tests (`RunDependencies.LimiterFactory`).
- For `qa` tasks, wrap the single call in a scheduler-backed execution path (workers=1).

Tests:
- `internal/runner/run_rate_limiter_test.go` (timeout <= 1s).

## Step 4: Concurrent question evaluation

- For `question_eval`, create a scheduler with task concurrency.
- Submit each question as a job; collect results by index.
- Add concurrency-safe verbose writers when workers > 1.
- Ensure task status calculation matches existing semantics.

Tests:
- `internal/runner/question_eval_concurrency_test.go` (timeout <= 2s).
- Update existing `question_eval` tests to be concurrency-safe.

## Step 5: Remote mode integration

- Ensure remote mode uses `httpclient` and respects request timeouts.
- Add `httptest`-based integration tests.

Tests:
- `internal/runner/remote_rate_limiter_test.go` (timeout <= 2s).

## Step 6: BDD scenarios

- Implement `testing.feature` scenarios with Godog steps.
- Use stubbed providers and local limits file for deterministic behavior.

Tests:
- Godog scenario timeouts <= 3s each.

## Step 7: Docs and examples

- Update README or example config to include `rate_limiter`.
- Add a sample limits file under `.cogni/` or `examples/` if needed.
