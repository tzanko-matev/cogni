# Concurrency Design (v1)

This document defines **how `question_eval` runs concurrently** while preserving deterministic results.

## Scope

- Only `question_eval` tasks run concurrently in v1.
- Other task types are out of scope; `question_eval` is the only supported task type.

## Execution model

### Sequential fallback

If `workers == 1`, keep the existing sequential loop for clarity.

### Concurrent path (workers > 1)

1) Load question spec.
2) Create a scheduler with `workers`.
3) For each question:
   - Build prompt.
   - Create a `ratelimiter.Job` with a buffered `resultCh`.
   - Submit job to the scheduler.
4) Wait for all `resultCh` values.
5) Store results in the original question order by index.
6) Shutdown scheduler with a timeout (2s).

## Job execution

Each `ratelimiter.Job.Execute`:

- Calls `call.RunCall` to execute the LLM request.
- Captures:
  - `callResult`
  - `runErr`
  - Parsed answer + correctness
- Sends a `QuestionResult` to `resultCh`.
- Returns `uint64(callResult.Metrics.Tokens)` as `actualTokens`.

Use a buffered channel (`size=1`) to avoid goroutine leaks.

## Deterministic ordering

Each job knows its `index` in the question list:

```
results[index] = questionResult
```

This preserves output ordering regardless of completion order.

## Error handling

Match existing behavior:

- If `runErr == call.ErrBudgetExceeded`, mark task as `budget_exceeded`.
- Otherwise, `runtime_error`.
- Parsing failures set `ParseError` but do not stop other questions.

If **any** question yields `runtime_error`, overall task status is `error`.
If **any** question yields `budget_exceeded`, overall task status is `fail`.
Otherwise, `pass` only if all answers are correct.

## Logging

When `workers > 1`, wrap verbose writers in a mutex to keep lines intact.
Interleaving should never produce partial lines.

## Concurrency + rate limiting

The scheduler handles:

- `Reserve` retries on denials (uses `retry_after_ms` + jitter).
- Regenerating LeaseID after denial.
- Retrying on transport errors.

This means **concurrency is safe** even when limits are tight:
jobs block in the scheduler until the limiter allows them.
