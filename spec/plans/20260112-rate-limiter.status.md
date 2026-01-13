# Rate Limiter Implementation Status

Status: In progress

ID: 20260112-rate-limiter.status

Created: 2026-01-12

Linked plan: [spec/plans/20260112-rate-limiter.plan.md](/plans/20260112-rate-limiter.plan/)

## Current status
- Phase 6 in progress: benchmarks added; BDD, loadtest, docs pending.

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

## Next steps
- Finish Phase 6: e2e TB tests, benchmarks, load-test tool, and BDD godog suite.

## Latest test run
- 2026-01-12: `go test ./internal/agent/...` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/agent/...` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-12: `go test ./internal/registry ./internal/api` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/registry ./internal/api` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-12: `go test ./internal/backend/memory` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/backend/memory` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-12: `go test ./internal/api` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/api` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-13: `go test ./pkg/ratelimiter/...` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test ./internal/backend/tb ./internal/tbutil` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test ./internal/testutil ./cmd/ratelimiterd` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test -tags=integration ./internal/backend/tb` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test ./internal/tbutil` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test -tags=stress ./internal/stress` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test -tags=chaos,integration ./internal/chaos` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test -tags=integration ./internal/e2e` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-13: `go test ./internal/backend/memory ./internal/bench ./internal/testutil` (failed: Go 1.25 toolchain not available in environment).

## Relevant source files (current or planned)
- internal/agent/runner.go
- internal/agent/call/*
- internal/registry/*
- internal/api/*
- internal/backend/*
- internal/backend/memory/*
- internal/api/*
- pkg/ratelimiter/*
- internal/registry/*
- internal/backend/memory/*
- internal/backend/tb/*
- internal/api/*
- internal/tbutil/*
- internal/testutil/*
- cmd/ratelimiterd/*
- pkg/ratelimiter/*
- flake.nix
- README.md
- internal/stress/*
- internal/chaos/*
- internal/backend/memory/debug_snapshot.go
- internal/e2e/e2e_tb_integration_test.go
- internal/backend/memory/memory_backend_bench_test.go
- internal/bench/bench_http_test.go

## Relevant spec documents
- spec/features/rate-limiter/overview.md
- spec/features/rate-limiter/api.md
- spec/features/rate-limiter/backend-memory.md
- spec/features/rate-limiter/backend-tb.md
- spec/features/rate-limiter/client-lib.md
- spec/features/rate-limiter/test-suite.md
- spec/features/rate-limiter/testing.feature
- spec/features/rate-limiter/implementation-plan.md
