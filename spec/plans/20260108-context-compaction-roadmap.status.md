# Status: Context Compaction Roadmap

Date: 2026-01-08

## Plan
- Plan: `spec/plans/20260108-context-compaction-roadmap.plan.md`

## Scope
- Define and implement a Codex-style history compaction flow with summarization and configurable limits to prevent prompt growth.

## Relevant Specs & Notes
- `spec/engineering/builtin-agent.md`
- `spec/engineering/configuration.md`
- `spec/overview/project-summary.md`

## Relevant Code
- `internal/agent/compaction.go`
- `internal/agent/runner.go`
- `internal/agent/tokens.go`
- `internal/agent/openrouter.go`
- `internal/runner/run_task.go`

## Status
- State: DONE
- Progress:
  - Investigated current compaction flow (token-limit based, no summaries).
  - Identified integration points in runner and agent history handling.
  - Added compaction config schema + validation and documented config updates.
  - Updated builtin agent compaction docs to reflect soft/hard limits and summaries.
  - Implemented summary-aware compaction with soft limits, summary prompts, and tool retention policies.
  - Wired compaction into the run loop with verbose logging and summary insertion.
  - Added unit + integration tests for compaction behavior and summary insertion.
  - Added compaction metadata to results output (count + last summary size).
  - Added provider capability flags for remote compaction support.

## Next Actions
- None (DONE).
