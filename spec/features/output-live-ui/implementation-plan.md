# Implementation Plan (v1)

## Step 1: Event model + UI state

- Define a `QuestionEvent` type and a pure reducer in `internal/ui/live`.
- Implement a Bubble Tea model with a table/list view for questions.
- Tests: reducer transitions, tool sub-status, retry counts.

## Step 2: Runner event emission

- Add an optional `RunObserver` to `runner.RunParams`.
- Emit question events from `runQuestionTask` and `executeQuestionJob`.
- Add scheduler hooks for reserve/deny/error to emit waiting statuses.
- Tests: runner emits expected events in order.

## Step 3: Tool call activity

- Wrap `ToolExecutor` to emit tool start/finish events.
- Surface tool name and duration in question row status.
- Tests: tool event integration.

## Step 4: CLI integration

- Add `--ui=auto|live|plain` flag to `cogni run` and `cogni eval`.
- Detect TTY; default to live UI only in TTY and non-verbose.
- On completion, stop UI and print existing summary output.
- Tests: UI selection logic.

## Step 5: Docs + BDD

- Update any user-facing docs if needed.
- Add/adjust BDD scenarios under `spec/features/output-live-ui/testing.feature`.

## Completion

Mark this plan and status file DONE when all steps and tests are complete.
