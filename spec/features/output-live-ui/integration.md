# Integration Design (v1)

This document explains how to integrate the live UI without changing
evaluation logic.

## New UI package

Add a package to keep UI logic isolated (examples):

- `internal/ui/live`
- `internal/ui/console`

This package should own:

- The Bubble Tea model/state.
- A pure reducer for applying events to state.
- Rendering logic (Bubble Tea + Bubbles + Lip Gloss).

## Runner event stream

Introduce a small event interface that the runner can emit while executing.
This keeps UI code out of the core logic.

### Suggested event interface

```
type RunObserver interface {
  OnRunStart(runID string, cfg spec.Config, repo string)
  OnTaskStart(taskID string, task spec.TaskConfig)
  OnQuestionEvent(evt QuestionEvent)
  OnTaskEnd(taskID string, status string, reason *string)
  OnRunEnd(results runner.Results)
}
```

`QuestionEvent` should include:

- task id
- question index / id
- event type (queued, scheduled, reserving, waiting, running, parsing, done)
- optional detail fields (retry_after_ms, error, tool name, tool duration)

**Important**: event emission must never block task execution. Use a buffered
channel or drop events if the UI is slow.

## Where to emit events

### Runner / question eval

- After questions are loaded: emit `queued` for each question.
- Before submitting each job to scheduler: emit `scheduled`.
- Before `call.RunCall`: emit `running`.
- After `call.RunCall`: emit `parsing`.
- After parsing:
  - `correct` or `incorrect`
  - `parse_error` if parsing fails
  - `budget_exceeded` if `call.ErrBudgetExceeded`
  - `runtime_error` for any other error
- If the task fails before questions run, emit `skipped` for all.

### Scheduler events (rate limiter)

To reflect limiter backoff, add optional observer hooks to the scheduler:

- `OnReserveStart(jobID)`
- `OnReserveDenied(jobID, retryAfterMs, errorCode)`
- `OnReserveError(jobID, err)`
- `OnExecuteStart(jobID)`

Use these to map to:

- `reserving`
- `waiting_rate_limit` (RetryAfterMs)
- `waiting_limit_decreasing` (error code prefix `limit_decreasing:`)
- `waiting_limiter_error`

### Tool activity

Wrap the tool executor to emit tool events:

- `ToolStart(questionID, toolName)`
- `ToolFinish(questionID, toolName, duration, error)`

The tool call duration is already available via `tools.CallResult.Duration`.

## CLI behavior

Add a UI mode flag (recommended):

- `--ui=auto|live|plain`
  - `auto` (default): live UI when stdout is TTY and `--verbose` is false.
  - `live`: force UI; if not a TTY, print a warning and fall back to plain.
  - `plain`: always skip live UI.

When live UI is enabled, it owns stdout while tasks run. Once complete, stop
the UI and print the existing summary lines (run id, accuracy, paths).

## Thread safety

The live UI must handle concurrent events safely:

- Use a buffered channel to decouple runner events from the UI.
- The reducer must be pure and deterministic to ease testing.
