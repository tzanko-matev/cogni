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
- State: TODO
- Progress:
  - Investigated current compaction flow (token-limit based, no summaries).
  - Identified integration points in runner and agent history handling.
  - Added compaction config schema + validation and documented config updates.
  - Updated builtin agent compaction docs to reflect soft/hard limits and summaries.

## Next Actions
- Implement summary-aware compaction logic + summary prompt defaults.
- Wire compaction into run loop with soft/hard limits and verbose logging.
- Add unit/integration tests for summary insertion and tool output retention.
