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

## Next Actions
- Confirm compaction policy details (tool output retention + soft vs hard limits).
- Decide summary prompt location + format and whether to expose config overrides.

