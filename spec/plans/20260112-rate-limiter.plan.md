# Rate Limiter Implementation Plan

Status: In progress

ID: 20260112-rate-limiter

Created: 2026-01-12

Linked status: [spec/plans/20260112-rate-limiter.status.md](/plans/20260112-rate-limiter.status/)

## Goal
Implement the full rate limiter feature set (server, backends, client library, tests, docs) per the rate-limiter spec pack.

## Scope
- Refactor agent call pipeline to support CallHook integration points.
- Implement limit registry + admin API.
- Implement memory backend, TB backend, and ratelimiterd server.
- Implement client library (HTTP client, local client, scheduler, batcher).
- Implement full test suite (unit/integration/stress/chaos) and BDD features.
- Update dev shell and documentation/examples.

## Non-goals
- Authentication/authorization.
- Persistent lease metadata across restarts.

## Inputs and references
- spec/features/rate-limiter/*
- spec/architecture/decisions/0001-0014
- spec/features/rate-limiter/implementation-plan.md

## Plan conventions
- Keep files <200 lines and SRP.
- Every step includes tests with explicit timeouts.
- Use `.feature` + godog for user-visible behaviors.
- Use `jj` commits after each self-contained step and update status file.

## Phases

### Phase 0 - Call pipeline refactor (ADR 0013)
- Extract `RunCall` + `CallHook` into `internal/agent/call`.
- Preserve behavior and metrics.
- Tests: `go test ./internal/agent/...` (new hook tests, timeout <=2s each).

### Phase 0.5 - Dev environment update
- Add TigerBeetle to `flake.nix` dev shell and export `TB_BIN`.
- Document non-Nix fallback.
- Tests: `go test ./...` (sanity), no new tests.

### Phase 1 - Limit registry + admin API
- Implement registry persistence + state handling.
- Add admin endpoints and validation.
- Tests: registry unit tests (<=1s), HTTP handler tests (<=2s).

### Phase 2 - Memory backend
- Implement rolling + concurrency semantics and debt tracking.
- Inject clock + add DebugSnapshot (test build tag).
- Tests: memory backend unit tests (<=2s each).

### Phase 3 - HTTP server (ratelimiterd)
- Implement Reserve/Complete + batch endpoints.
- Wire registry + backend.
- Tests: handler tests for ordering + errors (<=2s each).

### Phase 4 - Client library
- Implement Limiter interface, HTTP client, local client.
- Implement Batcher and Scheduler.
- Tests: batcher + scheduler unit tests (<=3s each).

### Phase 5 - TigerBeetle backend
- Implement TB backend, submitter, retry heuristics.
- Tests: integration tests (<=10s each) guarded by `TB_BIN` and tags.

### Phase 6 - Test suite alignment
- Implement remaining tests in spec/features/rate-limiter/test-suite.md.
- Add stress/chaos tests with tags and timeouts.

### Phase 7 - BDD features
- Implement spec/features/rate-limiter/testing.feature scenarios via godog.
- Tests: `go test -tags=cucumber ./...` (or scoped).

### Phase 8 - Docs and examples
- Add README snippet and example config/limits file.
- Verify docs build if needed.
