# Rate Limiter Implementation Status

Status: In progress

ID: 20260112-rate-limiter.status

Created: 2026-01-12

Linked plan: [spec/plans/20260112-rate-limiter.plan.md](/plans/20260112-rate-limiter.plan/)

## Current status
- Phase 2 complete: in-memory backend implemented with deterministic tests.

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

## Next steps
- Phase 3: implement HTTP server reserve/complete endpoints and batch support.

## Latest test run
- 2026-01-12: `go test ./internal/agent/...` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/agent/...` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-12: `go test ./internal/registry ./internal/api` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/registry ./internal/api` (failed: repo requires go >= 1.25, local is 1.21.6).
- 2026-01-12: `go test ./internal/backend/memory` (failed: Go 1.25 toolchain not available in environment).
- 2026-01-12: `GOTOOLCHAIN=local go test ./internal/backend/memory` (failed: repo requires go >= 1.25, local is 1.21.6).

## Relevant source files (current or planned)
- internal/agent/runner.go
- internal/agent/call/*
- internal/registry/*
- internal/api/*
- internal/backend/*
- internal/backend/memory/*
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

## Relevant spec documents
- spec/features/rate-limiter/overview.md
- spec/features/rate-limiter/api.md
- spec/features/rate-limiter/backend-memory.md
- spec/features/rate-limiter/backend-tb.md
- spec/features/rate-limiter/client-lib.md
- spec/features/rate-limiter/test-suite.md
- spec/features/rate-limiter/testing.feature
- spec/features/rate-limiter/implementation-plan.md
