# Status: Verbose Log File Output

Date: 2026-01-07

## Plan
- Plan: `spec/plans/20260107-verbose-log-file.plan.md`

## Scope
- Add `--log <path>` for `cogni run` to write verbose logs to a file without changing LLM behavior.

## Relevant Specs & Notes
- `spec/roles/working/core-cli-runner-engineer/engineering/observability.md`
- `spec/roles/working/core-cli-runner-engineer/requirements/functional.md`
- Planned Godog coverage: `spec/features/output-verbose-log.feature`

## Relevant Code
- `internal/cli/run.go`
- `internal/runner/run.go`
- `internal/agent/verbose_output.go`
- `internal/agent/verbose_format.go`

## Status
- State: DONE
- Progress:
  - Plan and status created.
  - Agent verbose logging supports an additional log writer with full (post-tool-truncation) output.
  - Runner + CLI wiring for `--log` implemented, including CLI flag parsing and log file creation.
  - CLI flag parsing test covers `--log`.
  - Godog feature file and steps added for verbose log expectations.

## Next Actions
- None.

## DONE
- DONE on 2026-01-07.
