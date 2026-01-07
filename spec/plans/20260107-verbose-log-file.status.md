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
- State: NOT STARTED
- Progress: Plan and status created.

## Next Actions
- Implement `--log` flag and wire multi-writer for verbose output.
- Add minimal tests for log file creation and content.

## DONE
- Not done.
