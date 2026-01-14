# Live UI Design (v1)

This document specifies the layout, statuses, and behavior of the live UI.

## Layout

Top to bottom:

1) **Header**: run id, repo, elapsed time, limiter mode, workers, batch.
2) **Summary line**: counts by status (queued, waiting, running, done) and outcome
   counts (correct, incorrect, parse_error, budget_exceeded, runtime_error).
3) **Task panel**: current task info (task id, questions file, agent, model).
4) **Question list**: stable order (original question order).
5) **Footer**: last event line + help hints.

Example columns:

- QID (index or question id)
- Truncated question text
- Status (primary + optional tool sub-status)
- Elapsed or total duration
- Tokens (if known)
- Retry count (if any)

## Status model

Each question row has:

- **Primary status**: one of the states below.
- **Tool sub-status** (optional): shown only when a tool is active or just completed.
- **Retry count**: incremented on each limiter requeue.

### Primary statuses (live)

1) `queued` - question known but not yet submitted to the scheduler.
2) `scheduled` - submitted to scheduler, waiting for a worker slot.
3) `reserving` - scheduler worker attempting Reserve.
4) `waiting_rate_limit` - Reserve denied (RetryAfterMs).
5) `waiting_limit_decreasing` - Reserve denied with `limit_decreasing:*`.
6) `waiting_limiter_error` - Reserve error (transport or backend error).
7) `running` - `call.RunCall` executing.
8) `parsing` - parsing the answer after the model finishes.

### Primary statuses (terminal)

9) `correct`
10) `incorrect`
11) `parse_error`
12) `budget_exceeded`
13) `runtime_error`
14) `skipped` - task failed before questions ran (invalid file or no questions)

### Tool sub-status

Show as a suffix to the primary status while running a tool call:

- `tool:<name> running`
- `tool:<name> waiting` (optional, if a tool call is long-running)
- `tool:<name> done (1.2s)` (briefly shown before returning to `running`)

Only the most recent tool call should be shown in the row. Keep history in a
detail pane if one is added later.

## Row details

Each row should display:

- **ID**: `Q01`, `Q02`, etc. Use `question.id` when available.
- **Question text**: truncated to ~60-80 chars to avoid wrapping.
- **Status**: primary status + optional tool sub-status.
- **Time**: elapsed while running, total on completion.
- **Tokens**: show `n/a` until metrics are known.
- **Retries**: show as `retry:2` when non-zero.

## Colors

- When `--no-color` is set, use plain text only.
- When color is enabled:
  - Correct -> green, incorrect -> yellow, runtime_error -> red, budget_exceeded -> red.
  - Waiting statuses -> cyan, running -> blue, parsing -> magenta.

## Behavior

- Refresh at a steady rate (5-10 FPS) and update only changed rows.
- Keep ordering stable to avoid visual jitter.
- On completion, stop the live UI and print the normal summary lines.

## Fallback

If stdout is not a TTY, or `--verbose` is used, skip the live UI and use
current plain output behavior.
