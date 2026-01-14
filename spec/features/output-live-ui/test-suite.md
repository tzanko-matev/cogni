# Test Suite (v1)

All tests should have explicit timeouts (<= 1s unless noted).

## Unit tests

### 1) UI reducer transitions

Package: `internal/ui/live` (or equivalent)

Verify state transitions for a single question:

- `queued -> scheduled -> reserving -> running -> parsing -> correct`
- `waiting_rate_limit` increments retry count and stores retry_after_ms
- `waiting_limit_decreasing` renders the proper status
- `runtime_error` and `budget_exceeded` map to terminal status
- `parse_error` preserves the parse error message
- tool sub-status updates on tool start/finish

Timeout: 1s per test.

### 2) Tool call activity

Simulate tool start and finish events:

- Row shows `tool:<name> running`
- Row shows `tool:<name> done (duration)` briefly or in last event line

Timeout: 1s per test.

## Integration tests

### 3) Runner event emission order

Create a fake observer and run a small question_eval task:

- Expect `queued` -> `scheduled` -> `reserving` -> `running` -> `parsing` -> `done`
- Ensure events are emitted for each question index.

Timeout: 2s.

### 4) Rate limiter backoff event

Use a stub limiter that denies the first Reserve and allows the second:

- Expect a `waiting_rate_limit` event with retry_after_ms > 0.

Timeout: 2s.

### 5) Tool events integration

Use a tool executor that sleeps briefly to simulate a long call:

- Expect `tool_start` and `tool_finish` for the question.

Timeout: 2s.

## CLI tests

### 6) UI selection logic

Add a test seam for TTY detection and UI mode selection:

- `--ui=plain` -> no live UI
- `--ui=auto` + non-TTY -> no live UI
- `--ui=auto` + TTY + not verbose -> live UI
- `--verbose` always disables live UI

Timeout: 1s per test.

## BDD scenarios

Use `spec/features/output-live-ui/testing.feature` to describe behavior:

- Live UI appears in TTY when not verbose.
- Live UI shows tool activity for a tool call.
- Non-TTY output falls back to plain summary.
