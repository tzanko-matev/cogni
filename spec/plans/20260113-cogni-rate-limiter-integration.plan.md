# Plan: Cogni rate limiter integration (2026-01-13)

## Goal
Implement spec/features/cogni-rate-limiter-integration end-to-end: config schema + validation, limiter construction, runner integration with concurrency, remote mode tests, BDD scenarios, and docs/examples.

## References
- spec/features/cogni-rate-limiter-integration/overview.md
- spec/features/cogni-rate-limiter-integration/config.md
- spec/features/cogni-rate-limiter-integration/integration.md
- spec/features/cogni-rate-limiter-integration/concurrency.md
- spec/features/cogni-rate-limiter-integration/test-suite.md
- spec/features/cogni-rate-limiter-integration/testing.feature

## Steps
1) Config schema + defaults + validation
   - Add `RateLimiterConfig` + `BatchConfig` to `internal/spec`.
   - Add `TaskConfig.Concurrency`.
   - Normalize defaults for rate limiter and batch settings.
   - Validate rate limiter config and task concurrency constraints.
   - Tests: `internal/config/config_rate_limiter_test.go` (timeout <= 1s per test).

2) Limiter construction helpers
   - Add `internal/ratelimit` package with `BuildLimiter`, `ResolveTaskWorkers`, `MaxOutputTokens`.
   - Add `ratelimiter.NoopLimiter`.
   - Add `httpclient.NewWithTimeout`.
   - Wrap limiter with batcher when enabled.
   - Tests: `internal/ratelimit/limiter_test.go` (timeout <= 1s), `pkg/ratelimiter/httpclient` timeout constructor tests (timeout <= 1s).

3) Runner wiring for rate limiting
   - Build limiter once per run and inject into task execution.
   - Add `RunDependencies.LimiterFactory` seam for tests.
   - Tests: `internal/runner/run_rate_limiter_test.go` (timeout <= 1s).

4) Concurrent question evaluation
   - Use per-task scheduler with worker count.
   - Run question_eval jobs concurrently, preserve deterministic ordering.
   - Guard verbose writers with a lock when workers > 1.
   - Tests: `internal/runner/question_eval_concurrency_test.go` (timeout <= 2s) and update existing question_eval tests if needed.

5) Remote mode integration
   - Verify HTTP client timeout usage and remote limiter path.
   - Tests: `internal/runner/remote_rate_limiter_test.go` (timeout <= 2s).

6) BDD scenarios
   - Implement `spec/features/cogni-rate-limiter-integration/testing.feature` with godog steps.
   - Tests: new cucumber suite under `tests/` with per-scenario timeout <= 3s.

7) Docs/examples
   - Update README or add example config/limits file to demonstrate `rate_limiter` block.

## Completion
Mark this plan and the status file as DONE when all steps and tests are complete.
