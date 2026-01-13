# Test Suite (v1)

All tests must use explicit timeouts.

## Unit tests

### 1) Config validation (`internal/config`)

File: `internal/config/config_rate_limiter_test.go`

- Missing `base_url` in remote mode => validation error.
- Missing `limits_path` in embedded mode => validation error.
- Invalid `mode` => validation error.
- `workers <= 0` => validation error.
- `batch.size <= 0` or `batch.flush_ms <= 0` => validation error.
- `task.concurrency <= 0` => validation error.
- `task.concurrency` on non-`question_eval` => validation error.

Timeout: 1s per test (use `testutil.Context(t, 1*time.Second)`).

### 2) Limiter construction (`internal/ratelimit`)

File: `internal/ratelimit/limiter_test.go`

- Disabled mode returns NoopLimiter.
- Embedded mode loads limits from file and succeeds with valid JSON.
- Embedded mode fails with missing file.
- Remote mode constructs HTTP client with timeout.
- Batcher wraps when `batch.size > 1`.

Timeout: 1s per test.

### 3) Scheduler concurrency (`internal/runner`)

File: `internal/runner/question_eval_concurrency_test.go`

Create a fake provider that:
- Blocks until two calls are started.
- Sleeps for a fixed duration (e.g., 150ms).

Assertions:
- With workers=2 and limiter disabled, total runtime < 300ms.
- With workers=1, total runtime >= 300ms.

Timeout: 2s per test.

### 4) Rate limiter usage (`internal/runner`)

File: `internal/runner/run_rate_limiter_test.go`

Use a stub limiter that records Reserve/Complete calls.

- `qa` task: one Reserve + one Complete per attempt.
- `question_eval` task: Reserve/Complete per question.

Timeout: 1s per test.

## Integration tests (local)

### 5) Remote mode against httptest server

File: `internal/runner/remote_rate_limiter_test.go`

Spin up `httptest.Server` with:
- `/v1/reserve` and `/v1/complete` handlers that always allow.

Run a `question_eval` task with `rate_limiter.mode=remote` and assert:
- Requests hit the server.
- Task completes successfully.

Timeout: 2s per test.

## BDD tests

Use `spec/features/cogni-rate-limiter-integration/testing.feature`.

Add Godog step definitions in `tests/` or `internal/cli` as appropriate.

Timeout per scenario: <= 3s.
