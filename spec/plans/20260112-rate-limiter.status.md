# Rate Limiter Implementation Status

Status: DONE

ID: 20260112-rate-limiter.status

Created: 2026-01-12

Linked plan: [spec/plans/20260112-rate-limiter.plan.md](/plans/20260112-rate-limiter.plan/)

## Current status
- Implementation complete; verification done across default, integration, stress+integration, chaos+integration, and cucumber suites with `TB_INTEGRATION=1`.

## What was done so far
- Created plan and status files for the rate limiter implementation.
- Added `internal/agent/call` package with `RunCall`, `CallHook`, and supporting helpers.
- Moved verbose formatting/logging helpers into `internal/agent/call`.
- Updated runner paths to use `call.RunCall` and adjusted agent tests accordingly.
- Added RunCall hook unit tests with explicit timeouts.
- Added TigerBeetle to `flake.nix` dev shell and auto-exported `TB_BIN`.
- Documented the non-Nix `TB_BIN` fallback in `README.md`.
- Added `pkg/ratelimiter` types and `internal/backend` interface.
- Implemented `internal/registry` with atomic save/load and state transitions.
- Implemented admin API handlers for PUT/GET limits and added validation/decrease tests.
- Implemented in-memory backend with rolling/concurrency limits, debt tracking, and decrease handling.
- Added memory backend unit tests using FakeClock, plus test helpers and stress coverage.
- Added FakeClock utilities in `internal/testutil`.
- Implemented reserve/complete + batch HTTP handlers with validation and ordering tests.
- Added LLM requirement helper and ULID generator in `pkg/ratelimiter`.
- Implemented client-side batcher and scheduler with deterministic unit tests.
- Added `internal/tbutil` helpers (ID derivation, client pool, submitter).
- Implemented TigerBeetle backend core (apply, reserve, complete, retry policy, decrease loop).
- Added ratelimiterd config + main wiring for memory/TB backends and healthz.
- Added test utilities (Eventually, ULID, StartTigerBeetle, StartServer, HTTP helpers).
- Fixed submitter batching to avoid oversize batches.
- Added TB backend integration tests (integration tag).
- Added submitter microbatch tests with explicit timeouts.
- Added stress tests for memory/TB backends and chaos tests for TB/server restarts.
- Enabled memory DebugSnapshot in stress builds for invariant checks.
- Added TB end-to-end integration suite with scheduler and batch coverage.
- Added memory backend and HTTP benchmark suites.
- Generalized testutil helpers to support benchmarks (testing.TB).
- Implemented godog BDD suite for rate limiter feature scenarios.
- Added cucumber build tag support for memory debug snapshots.
- Implemented ratelimiter load-test CLI tool.
- Added example ratelimiterd config + limits file and documented usage.
- Added validation for load-test configuration flags.
- Tightened stress test timeouts and bounded randomized workload iterations.
- Upgraded TigerBeetle Go SDK to v0.16.67 and migrated imports/API usage (Uint128 conversions, context wrappers).
- Added TB backend debug helper for pending debits to support stress assertions.
- Reworked stress stop signaling to broadcast cancellation and added TB pending debit polling for concurrency invariants.
- Added memory-backend registry attachment so applied decreases update registry state.
- Wired memory backend registry attachment in ratelimiterd, ratelimitertest servers, and cucumber harness.
- Fixed chaos TB test server helper signature to use ratelimitertest server type.

## Next steps
- None (implementation complete).

## Latest test run
- 2026-01-13: `nix develop -c go test ./...` (pass).
- 2026-01-13: `TB_INTEGRATION=1 nix develop -c go test -tags=integration ./...` (pass).
- 2026-01-13: `TB_INTEGRATION=1 nix develop -c go test -tags=stress,integration ./...` (pass).
- 2026-01-13: `TB_INTEGRATION=1 nix develop -c go test -tags=chaos,integration ./...` (pass).
- 2026-01-13: `nix develop -c go test -tags=cucumber ./...` (pass).

## Relevant source files (current or planned)
- internal/agent/runner.go
- internal/agent/call/*
- internal/registry/*
- internal/api/*
- internal/backend/*
- internal/backend/memory/*
- internal/backend/tb/*
- internal/tbutil/*
- internal/testutil/*
- cmd/ratelimiterd/*
- cmd/ratelimiter-loadtest/main.go
- pkg/ratelimiter/*
- internal/stress/*
- internal/chaos/*
- internal/e2e/e2e_tb_integration_test.go
- internal/bench/bench_http_test.go
- tests/ratelimiter/ratelimiter_cucumber_test.go
- flake.nix
- README.md
- go.mod
- go.sum

## Relevant spec documents
- spec/features/rate-limiter/overview.md
- spec/features/rate-limiter/api.md
- spec/features/rate-limiter/backend-memory.md
- spec/features/rate-limiter/backend-tb.md
- spec/features/rate-limiter/client-lib.md
- spec/features/rate-limiter/test-suite.md
- spec/features/rate-limiter/testing.feature
- spec/features/rate-limiter/implementation-plan.md
