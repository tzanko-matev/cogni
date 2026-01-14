# Plan: Embedded inline limits config (2026-01-14)

## Goal
Allow embedded rate limiter config to accept inline `limits` (same schema as limits.json) while keeping `limits_path`. Enforce exactly one of `limits` or `limits_path` in embedded mode.

## References
- spec/features/cogni-rate-limiter-integration/overview.md
- spec/features/cogni-rate-limiter-integration/config.md
- spec/features/cogni-rate-limiter-integration/integration.md
- spec/features/cogni-rate-limiter-integration/test-suite.md
- spec/features/cogni-rate-limiter-integration/testing.feature

## Steps
1) Config schema + validation for inline limits
   - Add `RateLimiterConfig.Limits []ratelimiter.LimitState` with `yaml:"limits"` and ensure ratelimiter limit structs accept snake_case YAML keys (mirror limits.json schema).
   - Update rate limiter validation to require exactly one of `limits` or `limits_path` in embedded mode.
   - Update config validation tests for missing both, both present, and inline limits accepted.
   - Tests: `nix develop -c go test ./internal/config`.

2) Embedded limiter construction from inline limits
   - Add a local limiter constructor that accepts limit states directly.
   - Update `internal/ratelimit.BuildLimiter` to prefer inline `limits` when present, else load from `limits_path` (resolved against repo root).
   - Extend limiter tests to cover inline limits and retain file-based behavior.
   - Tests: `nix develop -c go test ./internal/ratelimit ./pkg/ratelimiter/local`.

3) Docs + BDD coverage
   - Update config/integration/test-suite specs to document `limits` and the embedded validation rule.
   - Update or add example config showing inline limits in `.cogni/config.yml`.
   - Extend the Cogni rate limiter feature scenarios/steps to cover embedded inline limits.
   - Tests: `nix develop -c go test ./tests/cogni_rate_limiter_integration -tags=cucumber`.

## Completion
Mark this plan and the status file as DONE when all steps and tests are complete.
