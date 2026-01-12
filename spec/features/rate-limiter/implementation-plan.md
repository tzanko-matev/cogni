# Rate Limiter Implementation Plan (v1)

Each step must include tests with explicit timeouts. Use `.feature` files where behavior is user-visible.

## Step 0: Refactor LLM call pipeline (prerequisite)

- Follow ADR 0013 to extract `RunCall` and `CallHook` into `internal/agent/call`.
- Preserve current behavior and metrics.

Tests:

- Existing `go test ./internal/agent/...` with timeouts.
- Add unit tests for `RunCall` around hooks (timeout <= 2s per test).

## Step 1: Limit registry + admin API

- Implement registry (load/save JSON with atomic write).
- Persist LimitState (status + pending decrease) alongside definitions.
- Add admin endpoints:
  - `PUT /v1/admin/limits`
  - `GET /v1/admin/limits`
  - `GET /v1/admin/limits/{key}`
  - support capacity decrease state transitions

Tests:

- Unit tests for registry load/save (timeout <= 1s).
- HTTP handler tests for validation and unknown keys (timeout <= 2s).
- HTTP handler tests for capacity decrease state (timeout <= 2s).

## Step 2: In-memory backend

- Implement `MemoryBackend` with rolling and concurrency limits.
- Support debt tracking for overage.

Tests:

- Unit tests for reserve/deny/reconcile/overage (timeout <= 2s).

## Step 3: HTTP server (`ratelimiterd`)

- Implement `/v1/reserve`, `/v1/complete` and batch variants.
- Wire registry + backend.

Tests:

- Handler tests for batch ordering and per-item errors (timeout <= 2s).

## Step 4: Client library

- Implement Limiter interface (HTTP client + local client).
- Implement Batch client and Scheduler.

Tests:

- Unit tests for batcher flush behavior (timeout <= 2s).
- Unit tests for scheduler queue fairness (timeout <= 3s).

## Step 5: TigerBeetle backend

- Implement TB account provisioning.
- Implement Reserve/Complete with linked pending transfers.
- Implement microbatch submitter.
- Implement retry-after heuristics.
- Implement capacity decrease blocking and apply loop.

Tests:

- Integration test: linked transfers are atomic (timeout <= 10s).
- Integration test: `id_already_failed` behavior (timeout <= 10s).
- Integration test: overage debt tracking path (timeout <= 10s).
- Integration test: capacity decrease blocks new reservations until applied (timeout <= 10s).

## Step 6: BDD features

- Implement `spec/features/rate-limiter/testing.feature` scenarios.
- Ensure all scenarios specify timing expectations.

## Step 7: Docs and examples

- Add README snippet for ratelimiterd usage.
- Add example config and limits file.
