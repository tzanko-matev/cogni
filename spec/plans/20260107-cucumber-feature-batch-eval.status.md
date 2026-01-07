# Cucumber Feature-Batch Evaluation Status

Status: Done

ID: 20260107-cucumber-feature-batch-eval.status

Created: 2026-01-07

Linked plan: [spec/plans/20260107-cucumber-feature-batch-eval.plan.md](/plans/20260107-cucumber-feature-batch-eval.plan/)

## Current status
- Phases 0-5 complete: batch per-feature evaluation implemented, cleanup done, tests passing.

## Clarifications
- Each feature file triggers one agent run, which may include multiple LLM turns.
- Scenarios/examples are not evaluated via separate agent runs.

## What was done so far
- Created the plan and status entries for per-feature evaluation.
- Renamed the plan/status files to the dated naming format and removed the old filenames.
- Committed the updated AGENTS.md work process guidance (no change to plan scope).
- Updated cucumber evaluation design, API, data model, configuration, and requirements docs for per-feature batch prompts and responses.
- Updated cucumber feature specs to reflect per-feature agent runs and batch response validation.
- Added feature-level metrics (`feature_runs`) to cucumber results.
- Implemented batch agent response parsing + validation (missing/extra/duplicate -> error).
- Refactored cucumber runner to run one agent call per feature file and record feature-level metrics.
- Updated unit tests for batch parsing/validation and runner behavior; updated e2e cucumber prompt template.
- Removed per-example effort fields from cucumber example results (moved to feature runs).
- Ran `nix develop -c go test ./...` (2026-01-07).

## Next steps
- None. Task complete.

## Latest test run
- `nix develop -c go test ./...` (2026-01-07): pass.

## Relevant source files (current or planned)
- internal/runner/cucumber.go
- internal/cucumber/agent.go
- internal/runner/results.go
- internal/cli/run.go
- internal/config/validate.go
- internal/spec/types.go

## Relevant spec documents
- spec/design/cucumber-evaluation.md
- spec/design/api.md
- spec/design/data-model.md
- spec/engineering/configuration.md
- spec/requirements/functional.md
- spec/requirements/acceptance-criteria.md
- spec/features/cucumber-adapter-godog.feature
- spec/features/cucumber-adapter-manual.feature
