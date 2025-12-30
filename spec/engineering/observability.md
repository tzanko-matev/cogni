# Observability

## Logging

- CLI summary output for each run.
- Optional per-task logs in the run output directory.
- Verbose console logs for `cogni run --verbose` (LLM input/output, tool calls and results, per-task metrics; respect truncation limits), with ANSI styling when stdout is a terminal and `--no-color` for plain text.

## Metrics

- Tokens, wall time, tool calls, and files read per task.

## Tracing

- Not required in MVP.

## Alerts

- Not required in MVP.
