# Plan: Verbose Log File Output

Date: 2026-01-07

## Goal
Add a `--log <path>` flag to `cogni run` that writes verbose run logs to a file **without changing LLM behavior**. The log should capture the same (possibly truncated) content that the LLM actually sees.

## Non-Goals
- No change to agent prompt construction, tool execution, or truncation policies.
- No changes to task execution results or evaluation logic.
- No log rotation, compression, or multi-run aggregation.

## Current Behavior (Summary)
- `--verbose` streams LLM input/output, tool calls/results, and metrics to stdout.
- Verbose output is truncated (max bytes + tool output max lines) and tool outputs are truncated by runner limits.

## Proposed Behavior
- Add `--log <path>` flag to `cogni run`.
- When `--log` is set:
  - Verbose logs are written to the provided file path.
  - The file receives the same verbose content (including truncation) as the LLM sees.
  - `--verbose` can still be used to also stream to stdout.

## Godog Feature Coverage
Feature file: `spec/features/output-verbose-log.feature`

Scenarios to implement:
1. **Log file is written when --log is set**
   - Run `cogni run --log run.log`
   - Assert `run.log` exists and contains verbose markers (`[verbose]`).
2. **Log file captures verbose output without changing stdout**
   - Run `cogni run --log run.log`
   - Assert stdout still contains the normal run summary (no extra verbose-only noise).
3. **Log file and stdout both receive verbose output when --verbose and --log are set**
   - Run `cogni run --verbose --log run.log`
   - Assert stdout contains verbose markers and `run.log` contains the same markers.

## Implementation Steps
1. **CLI Flag & Wiring**
   - Add `--log` flag to `cogni run`.
   - Open/create the log file when provided.
   - Route verbose output to:
     - stdout when `--verbose` is set,
     - log file when `--log` is set,
     - both when both flags are set.
2. **Runner Interface**
   - Extend `runner.RunParams` to accept a `VerboseWriter` (already exists) and possibly a `VerboseWriters` slice or a helper to write to multiple outputs.
   - Ensure existing behavior remains unchanged when `--log` is not set.
3. **Logging Semantics**
   - Use the existing verbose formatting and truncation paths (no new truncation policy).
   - Confirm log output is the same content as console verbose (just redirected).
4. **Validation & Tests**
   - Add a CLI test to ensure `--log` is accepted.
   - Implement the Godog feature file and scenarios above.
   - Add/extend step definitions to assert log file existence and contents.

## Files to Touch
- `internal/cli/run.go` (flag parsing, log writer setup)
- `internal/runner/run.go` (propagate verbose writer / multi-writer)
- Possibly a small helper for multi-writer creation
- Tests under `internal/cli` or `internal/runner`

## Risks
- Accidental behavior change if tool output limits are modified. Avoid by keeping truncation untouched.

## Done Criteria
- `cogni run --log path.log` writes verbose logs to `path.log`.
- `cogni run --verbose --log path.log` writes to both stdout and file.
- No change in LLM inputs or outputs aside from logging destination.

## DONE
- Done on 2026-01-07.
