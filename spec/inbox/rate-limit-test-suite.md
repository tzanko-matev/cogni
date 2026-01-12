# Rate Limiter Test Suite Specification (Go)

This document specifies the **full** test suite we must implement for the rate-limiter (server + client library + backends). This component is critical, so we test:

* **Correctness** (single-threaded and concurrent)
* **Correctness under load** (high concurrency, long-running)
* **Performance** (latency, throughput, allocations)
* **Resilience** (retries, idempotency, partial failures)
* **TigerBeetle-specific semantics** (linked reservations, timeouts, and retry behavior)

The suite is intentionally extensive. Implement exactly what’s described and treat failing tests as “must fix in code”.

---

## 0) Scope (what we are testing)

We have these components:

1. **Limiter interface** used by clients (both local and remote):

* `Reserve(lease_id, job_id, requirements[]) -> allowed/denied + retry_after`
* `Complete(lease_id, job_id, actuals[]) -> ok`

2. **Backends**:

* **Memory backend** (single-binary deployments)
* **TigerBeetle backend** (distributed deployments; clients talk to Go server, server talks to TB)

3. **HTTP server** (`ratelimiterd`) exposing:

* `POST /v1/reserve`
* `POST /v1/complete`
* Admin limit registry:

  * `PUT /v1/admin/limits`
  * `GET /v1/admin/limits`
  * `GET /v1/admin/limits/{key}`

4. **Client library**:

* HTTP client (`pkg/ratelimiter/httpclient`)
* Local client (`pkg/ratelimiter/local`)
* Scheduler (prevents head-of-line blocking; retries denied requests with **new LeaseIDs**)

---

## 1) Core invariants (what must always be true)

These are the “truths” we assert repeatedly throughout tests:

### 1.1 Safety (never exceed limits)

For each limit key:

* Rolling limits (RPM/TPM/budget):

  * At any time, the sum of amounts of **active reservations** must be `<= capacity`.
* Concurrency limits:

  * At any time, active holds must be `<= capacity`.

### 1.2 Atomicity

A reserve request with multiple requirements must be **all-or-nothing**:

* If any requirement can’t be reserved, **none** of them are reserved.

### 1.3 Idempotency (critical for correctness with retries)

* If `Reserve` is called multiple times with the **same** `lease_id` and **same** requirements:

  * the result must be the same (`allowed` or `denied`) and it must **not** double-count capacity.
* `Complete` can be called multiple times for the same lease and must be a safe no-op after the first.

### 1.4 Retry semantics (TigerBeetle-specific; must be tested)

After a **denied** reservation attempt, retrying later with the **same** lease must not be required to succeed. Instead, clients must retry with a **new LeaseID**.

Therefore:

* Scheduler must generate **new LeaseIDs** on retry-after-deny.
* Tests must prove that “retry denied with same lease” does not become allowed later in TB mode (this validates our design choice).

### 1.5 Progress (no head-of-line blocking)

The scheduler must make progress when some work classes are denied:

* Jobs that could be allowed must not be blocked behind denied jobs.

### 1.6 Time correctness (expiry)

Rolling reservations must eventually expire and capacity returns.
Concurrency holds must be released on `Complete` or via timeout.

**Important:** TigerBeetle expiry cleanup is best-effort; tests must use **eventually/polling**, not exact timing.

---

## 2) Test organization and how to run

### 2.1 Build tags

We separate tests into categories so CI can run them appropriately.

* **Default (unit)**: `go test ./...`
* **Integration (TigerBeetle required)**: `go test -tags=integration ./...`
* **Stress (heavy randomized concurrency)**: `go test -tags=stress ./...`
* **Chaos (process kill / restart)**: `go test -tags=chaos ./...`

### 2.2 Standard commands (must work)

Add a `Makefile` with:

* `make test` → `go test ./...`
* `make test-race` → `go test -race ./...`
* `make test-integration` → `go test -tags=integration ./...`
* `make test-stress` → `go test -tags=stress -race ./...`
* `make test-chaos` → `go test -tags=chaos ./...`
* `make bench` → `go test -bench . -benchmem ./...`

### 2.3 External dependency for integration tests

Integration tests require the `tigerbeetle` binary.

* Environment variable: `TB_BIN=/path/to/tigerbeetle`
* If `TB_BIN` is missing, integration tests must `t.Skip()` with a clear message.

---

## 3) Required test utilities (implement first)

Create package: `internal/testutil`

### 3.1 Eventually helper

```go
func Eventually(t *testing.T, timeout, interval time.Duration, fn func() bool, msgAndArgs ...any)
```

* Poll `fn()` every `interval` until it returns true or `timeout` passes.
* On timeout, fail the test and print a useful message.

### 3.2 ULID generator

Add `NewULID()` returning string ULID (use any library; choose one and stick to it).

### 3.3 Fake clock (required for memory backend correctness tests)

Create:

```go
type FakeClock struct { ... }
func NewFakeClock(start time.Time) *FakeClock
func (c *FakeClock) Now() time.Time
func (c *FakeClock) Advance(d time.Duration)
```

**Requirement:** memory backend must accept an injected clock (interface) so tests can advance time without sleeping.

### 3.4 Start TigerBeetle (integration)

Implement:

```go
type TBInstance struct {
  ClusterID string   // "0"
  Addresses []string // ["127.0.0.1:PORT"]
  Stop func()
}

func StartTigerBeetleSingleReplica(t *testing.T) *TBInstance
```

Behavior:

* Pick a free TCP port.
* Create `t.TempDir()`
* Run:

  1. `tigerbeetle format ... --development <file>`
  2. `tigerbeetle start --addresses=<port> --development <file>`
* Wait until TCP port is accepting connections.
* Return instance and stop function (kill process, wait, cleanup).

All subprocesses must have:

* stdout/stderr captured and included in test failure output on error.

### 3.5 Start server (HTTP)

Implement:

```go
type ServerInstance struct {
  BaseURL string
  Close func()
}

func StartServer(t *testing.T, cfg ServerConfig) *ServerInstance
```

Use `httptest.NewServer(handler)` so tests get a real URL.

Provide helpers:

* `HTTPPutLimit(t, baseURL, def)`
* `HTTPGetLimit(t, baseURL, key)`
* `HTTPReserve(t, baseURL, req)`
* `HTTPComplete(t, baseURL, req)`

All HTTP calls must:

* use `context.WithTimeout(ctx, 2*time.Second)`
* fail tests on non-2xx (unless the test expects it).

---

## 4) Unit tests (fast, deterministic, always run)

Create these files and tests. Names are mandatory.

### 4.1 Registry tests (`internal/registry/registry_test.go`)

1. `TestRegistry_RoundTrip_SaveLoad`

* Create registry with 3 limit defs.
* Save to temp file.
* Load from file.
* Assert deep-equal on definitions (order-insensitive).

2. `TestRegistry_AtomicWrite_NoTmpLeftBehind`

* Save registry.
* Assert `limits.json.tmp` does not exist after save.
* Assert JSON parses.

3. `TestRegistry_ConcurrentAccess_NoRace`

* Run only under `-race` but keep test in unit suite.
* Spawn 50 goroutines calling `Get(key)` in a loop.
* In parallel, update via `Put(def)` in a loop.
* Run for 250ms.
* Must not panic; race detector should be clean.

### 4.2 Client requirement builder tests (`pkg/ratelimiter/requirements_test.go`)

1. `TestBuildLLMRequirements_ContainsExpectedKeys`

* Input provider/model/tenant/prompt/max_output.
* Verify requirements include rpm/tpm/concurrency and optional daily budget.
* Verify TPM amount = `len(prompt bytes) + max_output_tokens`.

### 4.3 Scheduler tests (`pkg/ratelimiter/scheduler_test.go`)

Create a fake Limiter implementation:

* Allows all except keys with `provider=openai` (always denied with retry_after=100ms), or similar policy.

Tests:

1. `TestScheduler_NoHeadOfLineBlocking`

* Submit jobs in order: OpenAI (denied), Anthropic (allowed), Anthropic (allowed).
* Assert Anthropic jobs execute within 200ms (not blocked behind OpenAI).

2. `TestScheduler_RetryUsesNewLeaseID`

* Fake limiter denies first reserve for a job, then allows next attempt.
* Scheduler must generate a new LeaseID for the retry.
* Assert it called Reserve at least twice with **different** LeaseIDs before Execute runs.

3. `TestScheduler_CompleteAlwaysCalledAfterAllowed`

* Fake limiter always allows.
* Execute returns actual tokens.
* Assert Complete called once per allowed job.

---

## 5) Memory backend correctness tests (deterministic; no sleeps)

File: `internal/backend/memory/memory_backend_test.go`

All memory backend tests MUST use `FakeClock` and must not sleep.

### Setup helper

Implement:

```go
func newMemoryBackendForTest(t *testing.T, clock Clock) *MemoryBackend
```

And a helper to define limits quickly:

```go
func applyDefs(t *testing.T, b Backend, defs ...LimitDefinition)
```

### Tests (mandatory)

1. `TestMemory_Rolling_AllowThenDeny`

* Define `k1` rolling cap=2 window=10s.
* Reserve 2 leases → allowed.
* Third lease → denied.
* Assert `retry_after_ms > 0`.

2. `TestMemory_Rolling_ExpiryReleasesCapacity`

* cap=1 window=10s.
* Reserve once allowed.
* Advance clock by 11s.
* Reserve again allowed.

3. `TestMemory_MultiKeyAtomicity_NoPartialReserve`

* k1 cap=1 rolling
* k2 cap=0 rolling
* Reserve requiring both → denied.
* Immediately reserve requiring only k1 → allowed (proves k1 wasn’t consumed by failed multi-key attempt).

4. `TestMemory_Concurrency_ReleaseOnComplete`

* conc key cap=1 timeout=300s.
* Reserve lease1 (includes conc requirement) → allowed.
* Reserve lease2 → denied.
* Complete lease1 → ok.
* Reserve lease3 → allowed immediately.

5. `TestMemory_Concurrency_TimeoutReleases`

* conc cap=1 timeout=3s.
* Reserve lease1 allowed.
* Do NOT complete.
* Advance clock by 4s.
* Reserve lease2 allowed.

6. `TestMemory_ReconcileFreesSlack`

* rolling TPM key cap=100 window=10s.
* Reserve with upper=100 allowed.
* Complete with actual=10.
* Reserve another with amount=90 must be allowed without advancing time.

7. `TestMemory_ReserveIdempotent_NoDoubleCount`

* rolling cap=1 window=10s.
* Reserve lease1 allowed.
* Repeat Reserve lease1 with identical requirements: must return allowed (or same decision) and not consume more.
* Then Reserve lease2 must be denied until expiry.

8. `TestMemory_CompleteIdempotent_NoError`

* After allowed reserve, call Complete twice; second must be ok.
* Also test Complete for unknown lease_id must be ok (no-op), unless your implementation chooses to return a structured “not found” that callers can ignore. **Pick one behavior and codify it here** (recommended: ok/no-op).

9. `TestMemory_ApplyDefinition_IncreaseCapacityTakesEffectImmediately`

* rolling cap=1 → allow 1 then deny.
* update cap=2 via ApplyDefinition.
* Now second reserve must allow immediately.

10. `TestMemory_ApplyDefinition_DecreaseCapacity_DeniesUntilUsageDecays`

* rolling cap=2, reserve twice allowed.
* decrease cap=1.
* third reserve denied.
* advance clock until one expires.
* then reserve allowed (but still only one at a time).

11. `TestMemory_ConcurrentStress_NoRacesAndNeverExceeds`

* Run under `-race`.
* Define:

  * rolling rpm cap=50 window=2s
  * rolling tpm cap=2000 window=2s
  * conc cap=10 timeout=2s
* Spawn 100 goroutines for 500ms:

  * each loop creates new lease
  * Reserve with 3 keys (rpm=1, tpm=random 1..200, conc=1)
  * if allowed: advance fake clock a small random amount? (controlled) and Complete
* After run, assert backend internal invariants: used<=cap for all keys.

  * To do this cleanly, add `DebugSnapshot()` behind `//go:build test` that returns current used and reservations count per key.

---

## 6) HTTP API tests (memory backend)

File: `internal/api/http_api_test.go`

Start the server in-memory mode with `httptest.NewServer`.

### Tests

1. `TestHTTP_AdminPutGetList`

* PUT 3 limit definitions.
* GET list → contains all 3.
* GET by key → exact match.

2. `TestHTTP_Reserve_UnknownKeyReturns404`

* No definition exists.
* Reserve using a key → must return 404 (or 400).
  Pick one and enforce consistently (recommended: 404).

3. `TestHTTP_Reserve_ValidationErrorsReturn400`

* Missing lease_id
* Empty requirements
* amount=0
  Each must return 400.

4. `TestHTTP_ReserveAndComplete_EndToEnd`

* Define keys (rpm/tpm/conc).
* Reserve allowed.
* Complete ok.
* Make sure concurrency is released and another reserve can be allowed.

5. `TestHTTP_ReserveIdempotent`

* Same lease_id repeated:

  * First Reserve allowed
  * Second Reserve must be allowed too, with no extra consumption.

6. `TestHTTP_CompleteIdempotent`

* Complete twice for same lease; both 200.

---

## 7) TigerBeetle backend integration tests (requires TB)

All tests in this section must be under build tag `integration`.

### 7.1 TB backend direct tests

File: `internal/backend/tb/tb_backend_integration_test.go`
Guard: if `TB_BIN` missing → skip.

Start TB with `StartTigerBeetleSingleReplica(t)`.

#### Use short windows

To keep tests fast:

* rolling windows: 2s or 3s
* concurrency timeouts: 2s or 3s

#### Tests

1. `TestTB_ApplyDefinition_CanReserveImmediately`

* Apply rolling defs for rpm/tpm, and concurrency.
* Reserve with 3 requirements should return allowed.

2. `TestTB_MultiKeyAtomicity_LinkedAllOrNothing`

* Define k1 cap=1 rolling window=3s
* Define k2 cap=0 rolling window=3s
* Reserve requiring both → denied
* Reserve requiring only k1 → allowed
  This proves no partial reserve.

3. **Most important test**: `TestTB_DeniedAttemptMustUseNewLeaseID`
   This validates our LeaseID retry design.

Scenario:

* rolling key cap=1 window=2s
* Reserve leaseA → allowed
* Reserve leaseB → denied
* Wait until window passes + buffer (sleep 3s)
* Retry leaseB again:

  * expected: **still denied** (because the same transfer ID already failed)
* Retry with leaseC:

  * expected: allowed

Notes:

* This test will fail if the backend incorrectly allows reuse of a previously denied attempt.
* This test is the reason the scheduler must generate a new LeaseID on retries.

4. `TestTB_ReserveIdempotent_AllowedDoesNotDoubleCount`

* rolling cap=1 window=3s
* Reserve leaseA allowed
* Reserve leaseA again with same reqs: must return allowed (treat TB “exists” as success)
* Reserve leaseB must be denied until expiry

5. `TestTB_Concurrency_ReleasedOnComplete`

* conc cap=1 timeout=10s
* Reserve leaseA (includes conc) allowed
* Reserve leaseB denied
* Complete leaseA
* Reserve leaseC allowed immediately

6. `TestTB_Concurrency_TimeoutReleases`

* conc cap=1 timeout=2s
* Reserve leaseA allowed, do not complete
* sleep 3–4s
* Reserve leaseB allowed

7. `TestTB_ReconcileFreesSlack`

* rolling tpm cap=100 window=3s
* Reserve leaseA with amount=100 allowed
* Complete leaseA with actual=10 quickly
* Immediately attempt Reserve leaseB amount=90
* Use `Eventually` (timeout 2s, interval 50ms) to assert leaseB becomes allowed quickly

8. `TestTB_DynamicLimitCreation_NoRestartRequired`
   (Validates “add new limit at runtime” requirement.)

* Start TB
* Start server in TB mode with empty registry
* Admin PUT a new limit def with a new key
* Immediately Reserve using that key must work

9. `TestTB_CapacityIncrease_TakesEffectImmediately`

* cap=1, allow 1 then deny
* update capacity to 2 via ApplyDefinition (or admin endpoint)
* new reserve should allow immediately

10. `TestTB_CapacityDecrease_DeniesUntilUsageDecays`

* cap=2, reserve twice allowed
* update capacity to 1
* third reserve denied
* after one expires (sleep), reserve allowed again

### 7.2 Server+TB end-to-end tests

File: `internal/e2e/e2e_tb_integration_test.go` (tag: integration)

Start TB, then `ratelimiterd` with TB backend and real HTTP.

Tests:

1. `TestE2E_TB_AdminDefineThenReserve`
2. `TestE2E_TB_ReserveComplete_Idempotency`
3. `TestE2E_TB_Scheduler_NoHOL_WithRealServer`

* Use scheduler with HTTP client
* Define OpenAI limits so it’s saturated (cap=0 or very low)
* Define Anthropic higher
* Submit mixed jobs; Anthropic jobs must complete even if OpenAI jobs are denied.

---

## 8) Submitter/microbatcher tests (important for correctness + perf)

The TB server uses a microbatcher. We must test it heavily.

File: `internal/tbutil/submitter_test.go`

### Required behavior to test

1. **Never split a single work item** across multiple TB `create_transfers` calls.

   * If a work item does not fit within `max_batch_events`, the submitter must return an error for that item (“work item too large”) rather than splitting.

2. **Flush interval works**

   * A single work item submitted should complete within `flush_interval + small slack`.

3. **Max batch respected**

   * Submit many small work items; ensure individual TB calls do not exceed max batch size.

Tests:

1. `TestSubmitter_DoesNotSplitWorkItem`

* Configure max_batch_events = 5
* Create a work item with 6 transfers
* Expect: returned error immediately (no hang)

2. `TestSubmitter_FlushesWithinInterval`

* Configure flush interval = 5ms
* Submit a single small work item
* Ensure it completes within 50ms

3. `TestSubmitter_BatchSizeNeverExceedsMax`

* Submit 100 work items of 1 transfer
* Instrument submitter to record each flush batch size (add a test hook)
* Assert every batch size <= max

---

## 9) Stress tests (correctness under load)

These tests exist to prove correctness under concurrency and to catch races and subtle bugs.

### 9.1 Stress tests for memory backend (tag: `stress`)

File: `internal/stress/stress_memory_test.go`

Test: `TestStress_Memory_RandomizedWorkload`

* Use FakeClock but still run with real goroutines.
* Define limits for 3 “providers” with different caps.
* Run for 10 seconds:

  * 200 goroutines submit Reserve/Complete cycles with random:

    * provider/model selection
    * token upper bounds
    * completion duration
    * actual tokens (<= upper)
* Assertions:

  1. No panic, no deadlocks.
  2. Under `-race`, no races.
  3. Snapshot invariants at end: used<=cap per key.
  4. Progress: allowed_count > 0.

### 9.2 Stress tests for TB backend (tag: `stress,integration`)

File: `internal/stress/stress_tb_test.go`

Test: `TestStress_TB_RandomizedWorkload`

* Start TB.
* Start server in TB mode.
* Put limit defs (rolling and concurrency) with small windows (2–3s).
* Run 10 seconds:

  * 200 goroutines call server Reserve/Complete via HTTP client.
  * Use real sleeps for simulated LLM duration: random 0–50ms.
* Assertions:

  1. HTTP 500 rate must be 0 (or extremely low only if TB restarted; but we won’t restart in this test).
  2. Progress: allowed_count > 0.
  3. Concurrency invariant: track in-flight count per key based on allowed/completed events; max <= cap.

**Extra correctness check (optional but recommended):**

* Implement offline checker for RPM:

  * For each rpm key, collect timestamps of allowed reserves and verify no window has > cap.
  * Use a sliding window algorithm on timestamps.

---

## 10) Chaos/resilience tests (tag: `chaos,integration`)

File: `internal/chaos/chaos_tb_test.go`

These tests validate behavior when dependencies fail.

### 10.1 TB process kill and recovery

Test: `TestChaos_TBRestart_ServerRecovers`

* Start TB
* Start server TB mode
* Put limits
* Start load (50 goroutines calling reserve in loop)
* Kill TB process (Stop), keep load running for 2 seconds:

  * server should return errors quickly (no hangs)
* Restart TB on a new port (new TB instance)
* Reconfigure server if needed (best: server supports reconnect automatically; if not, restart server)
* Ensure within 5 seconds:

  * server again returns allowed responses
  * no deadlocks

### 10.2 Server restart does not break correctness

Test: `TestChaos_ServerRestart_InFlightReservationsExpire`

* Start TB + server
* Put limits (concurrency timeout=2s, rolling window=2s)
* Reserve a few leases and do not complete
* Restart server immediately
* Ensure after timeouts:

  * new reserves can be allowed again

---

## 11) Performance benchmarking (must implement)

### 11.1 Benchmarks for memory backend (always run)

File: `internal/backend/memory/memory_backend_bench_test.go`

Benchmarks:

1. `BenchmarkMemoryReserve_1Key`
2. `BenchmarkMemoryReserve_4Keys`
3. `BenchmarkMemoryComplete_4Keys`
4. `BenchmarkScheduler_Throughput_MemoryBackend`

Rules:

* Use `b.ReportAllocs()`
* Use FakeClock (no sleeps)
* Pre-apply limit definitions once

### 11.2 Benchmarks for HTTP server (optional but valuable)

File: `internal/bench/bench_http_test.go`

Benchmark:

* Start server in memory mode with httptest
* Benchmark Reserve via HTTP (1 key and 4 keys)
* This measures JSON + handler overhead.

### 11.3 TB performance harness (not a benchmark)

Create tool: `cmd/ratelimiter-loadtest`

Flags:

* `--mode=http|local`
* `--backend=memory|tigerbeetle`
* `--duration=30s`
* `--concurrency=200`
* `--providers=...`
* prints:

  * reserves/sec, completes/sec
  * p50/p95/p99 reserve latency
  * p50/p95/p99 complete latency
  * allowed/denied counts
  * server batch size stats (requires server metrics hook)

This tool is run manually and in nightly CI.

---

## 12) Test coverage checklist (definition → spend → replenish)

Every limit kind must be tested across the whole lifecycle:

### 12.1 Define/update

* Admin PUT works for new keys
* Update capacity up/down works
* New keys can be added at runtime in TB mode without restart

### 12.2 Spend

* Reserve allow/deny correctness
* Multi-key atomicity
* Idempotent Reserve

### 12.3 Replenish

* Rolling: expiry eventually restores capacity
* Concurrency: Complete releases immediately; timeout releases eventually
* Reconcile: frees slack early and increases throughput

---

## 13) Minimum CI plan (what must run automatically)

Because this is critical, CI must run:

**On every PR**

* `make test`
* `make test-race`
* `make test-integration` (single-replica TB)

**Nightly**

* `make test-stress` (includes integration+stress+race)
* Run `cmd/ratelimiter-loadtest` for 10 minutes and store output artifacts

---

## 14) Implementation notes for the junior dev

1. Implement `internal/testutil` first. Everything depends on it.
2. Implement memory backend tests next (they are the oracle and easiest to debug).
3. Implement HTTP tests with memory backend (validates API correctness).
4. Then implement TB integration tests (proves semantics and dynamic limit creation).
5. Finally implement stress/chaos/perf harness.

If you follow this order, failures will be easier to isolate.


This suite is intentionally “over-tested”. That’s the goal: the limiter is a critical dependency, so we want correctness guarantees, high confidence under load, and strong performance visibility.
