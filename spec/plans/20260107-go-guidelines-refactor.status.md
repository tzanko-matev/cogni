# Go Guidelines Refactor Status

Status: In progress

ID: 20260107-go-guidelines-refactor.status

Created: 2026-01-07

Linked plan: [spec/plans/20260107-go-guidelines-refactor.plan.md](/plans/20260107-go-guidelines-refactor.plan/)

## Current status
- Phase 0 in progress (inventory and boundaries).

## Clarifications
- Guideline enforcement applies to tests too (no `any`).
- All user-visible CLI behaviors should be covered by .feature + godog tests.
- Live-key tests can live in a separate suite, runnable via a single command.

## What was done so far
- Completed a repository-wide Go code review against AGENTS.md guidelines.
- Identified key violations (file size, `any`, missing docstrings, I/O mixing, test flakiness).
- Prepared a phased refactor plan with acceptance criteria.
- Updated plan with clarifications: tests must avoid `any`, all user-visible CLI behavior covered by godog, live-key suite via `just test-live`.
- Marked the plan status as in progress.

## Phase 0 notes (inventory)
### Files >200 lines (targets to split)
- internal/runner/run.go (513)
- internal/runner/cucumber.go (335)
- internal/agent/openrouter.go (346)
- internal/tools/runner.go (315)
- internal/eval/qa.go (277)
- internal/agent/verbose.go (269)
- internal/config/validate.go (260)
- internal/cucumber/expectations.go (203)
- tests/cucumber/steps_test.go (263)
- internal/runner/cucumber_test.go (250)
- internal/cli/e2e_test.go (644)

### `any` usage (must remove, including tests)
- internal/agent/session.go (ToolDefinition.Parameters, HistoryItem.Content)
- internal/agent/openrouter.go (openRouterFunctionDefinition.Parameters)
- internal/eval/qa.go (parsed JSON and citations handling)
- internal/runner/run.go (tool definitions via map[string]any)
- internal/agent/tool_executor.go (tool args maps)
- internal/cucumber/expectations.go (Examples any + maps)
- internal/agent/runner.go (ToolCall.Args)
- internal/agent/verbose.go (logVerbose args)
- internal/runner/output_writer.go (writeJSON)
- tests use `map[string]any` in runner/cucumber tests

### I/O boundaries to isolate
- Git commands: internal/vcs/git.go, internal/cli/e2e_test.go, internal/vcs/git_test.go
- rg commands: internal/tools/runner.go + internal/tools/runner_test.go
- HTTP/LLM: internal/agent/openrouter.go + internal/cli/e2e_test.go
- Filesystem reads: internal/runner/cucumber.go, internal/eval/qa.go, internal/tools/runner.go
- Exec: internal/runner/run.go (setup commands)

### Tests with external dependencies / no timeouts
- Live LLM: internal/cli/e2e_test.go (requireLiveLLM, http calls)
- Git binary: internal/cli/e2e_test.go, internal/vcs/git_test.go
- rg binary: internal/tools/runner_test.go
- No explicit timeouts in unit tests (many use context.Background without deadlines)

## Next steps
- Finish Phase 0 by defining module split boundaries and type replacement strategy.
- Begin Phase 1 docstrings + typing cleanup.

## Latest test run
- Not run in this update.

## Relevant source files (current or planned)
- internal/runner/run.go
- internal/runner/cucumber.go
- internal/agent/openrouter.go
- internal/agent/session.go
- internal/tools/runner.go
- internal/eval/qa.go
- internal/cli/e2e_test.go
- internal/vcs/git.go
- internal/vcs/git_test.go
- internal/tools/runner_test.go

## Relevant spec documents
- spec/engineering/testing.md
- spec/engineering/repo-structure.md
- spec/engineering/build-and-run.md
- spec/design/api.md
- spec/design/data-model.md
