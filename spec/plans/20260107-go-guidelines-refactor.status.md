# Go Guidelines Refactor Status

Status: In progress

ID: 20260107-go-guidelines-refactor.status

Created: 2026-01-07

Linked plan: [spec/plans/20260107-go-guidelines-refactor.plan.md](/plans/20260107-go-guidelines-refactor.plan/)

## Current status
- Phase 0 complete; Phase 1 in progress (typing + docstrings).

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

### Planned module boundaries
- internal/runner/run.go -> run_plan.go (planning + selection), run_execute.go (task execution), run_summary.go (summaries), run_tools.go (tool defs), run_paths.go (repo/output paths), run_setup.go (setup commands).
- internal/runner/cucumber.go -> cucumber_ground_truth.go, cucumber_prompt.go, cucumber_execute.go, cucumber_evaluate.go.
- internal/agent/openrouter.go -> openrouter_client.go (HTTP), openrouter_messages.go (message/tool build), openrouter_stream.go (stream parsing).
- internal/tools/runner.go -> runner_files.go (list/read), runner_search.go, runner_limits.go (output limiting), runner_paths.go (resolve/normalize).
- internal/eval/qa.go -> qa_parse.go (JSON parsing), qa_schema.go, qa_citations.go, qa_helpers.go.
- internal/agent/verbose.go -> verbose_format.go (format/indent), verbose_output.go (logging), verbose_palette.go.
- internal/config/validate.go -> validate_core.go (validation logic), validate_errors.go (error types), validate_rules.go (rule helpers).
- internal/cucumber/expectations.go -> expectations_parse.go, expectations_validate.go, expectations_types.go.
- tests/cucumber/steps_test.go -> steps_config.go, steps_run.go, steps_assert.go.
- internal/runner/cucumber_test.go -> cucumber_batch_test.go, cucumber_errors_test.go.
- internal/cli/e2e_test.go -> e2e_config_test.go, e2e_history_test.go, e2e_cucumber_test.go, e2e_outputs_test.go (plus live suite file with build tag).

### Type replacement strategy
- Replace `map[string]any` tool args with `map[string]json.RawMessage` + typed decoders per tool.
- Replace `HistoryItem.Content` with a sealed `HistoryContent` interface and concrete types (`HistoryText`, `HistoryToolCall`, `HistoryToolOutput`).
- Replace tool parameter schemas with explicit `ToolSchema` structs instead of `map[string]any`.
- For JSON parsing in eval/cucumber, use typed response structs or `json.RawMessage` + validators; avoid `any` in all packages and tests.

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

## Phase 1 progress
- Eliminated `any` usage across agent/eval/cucumber/runner output paths via typed JSON handling.
- Added typed tool schemas (`ToolSchema`), tool args (`ToolCallArgs`), and history content (`HistoryText`).
- Reworked expectations parsing to use `yaml.Node` instead of dynamic maps.
- Added docstrings across core agent runtime files.
- Added docstrings across eval and cucumber core modules.
- Added docstrings across runner, tools, config, report, vcs, and spec core modules.

## Next steps
- Finish Phase 0 by defining module split boundaries and type replacement strategy.
- Begin Phase 1 docstrings + typing cleanup.

## Latest test run
- `nix develop -c go test ./...` (2026-01-07): pass.

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
