# Go Guidelines Refactor Status

Status: In progress

ID: 20260107-go-guidelines-refactor.status

Created: 2026-01-07

Linked plan: [spec/plans/20260107-go-guidelines-refactor.plan.md](/plans/20260107-go-guidelines-refactor.plan/)

## Current status
- Phase 2 complete; Phase 3–4 in progress (I/O isolation + deterministic tests).
- Phase 1 nearing completion (remaining docstrings in tests/helpers).

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
- Split runner/cucumber and tools modules into smaller single-responsibility files (<200 lines).
- Split agent/openrouter, agent/verbose, eval/qa, config/validate, cucumber/expectations, agent/session, and cucumber/godog into smaller modules.
- Split oversized test files (runner cucumber tests, cucumber step definitions, CLI e2e tests).
- Added dependency injection for git/rg/http/filesystem boundaries and runner metadata resolution.
- Added testutil context timeouts; removed external rg/git dependencies from default tests.
- Tagged live LLM tests (`//go:build live`) and cucumber feature tests (`//go:build cucumber`); added `just test-live` and `just test-cucumber`.

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
- Added docstrings across CLI and command entrypoints.
- Confirmed `rg "\\bany\\b" -g '*.go'` shows no `any` usages outside vendor files.

## Phase 2 progress
- Split `internal/runner/run.go` into plan/setup/task/summary/tools/paths/types modules.
- Split `internal/runner/cucumber.go` into ground truth, grouping, prompt, and feature evaluation helpers.
- Split `internal/tools/runner.go` into constructor/list/search/read/paths/exec/output modules.
- Split `internal/agent/openrouter.go`, `internal/agent/verbose.go`, `internal/eval/qa.go`, `internal/config/validate.go`, `internal/cucumber/expectations.go`, `internal/agent/session.go`, `internal/cucumber/godog.go`.
- Split oversized test files into smaller units (<200 lines each).

## Phase 3 progress
- Introduced `gitRunner` and `Client` in `internal/vcs` for injectable git behavior.
- Added `rgRunner` and `fileSystem` abstractions to `internal/tools.Runner`.
- Added `HTTPDoer` for OpenRouter providers and `QADeps` filesystem abstraction for citation validation.
- Added runner dependency hooks for repo root and metadata to avoid git in unit tests.

## Phase 4 progress
- Replaced rg and git usage in unit tests with fakes.
- Added `testutil.Context` helper to enforce per-test context deadlines.
- Tagged live LLM and cucumber feature suites; added Justfile targets to run them.

## Next steps
- Finish docstrings for remaining tests/helpers and any missed functions.
- Continue isolating remaining I/O boundaries (setup commands, godog exec path, feature reads) into injectable adapters.
- Ensure remaining tests use explicit timeouts or testutil helpers consistently.
- Verify live/cucumber suites with `just test-live` / `just test-cucumber`.

## Latest test run
- `nix develop -c go test ./...` (2026-01-07): pass.

## Relevant source files (current or planned)
- internal/runner/run.go
- internal/runner/run_plan.go
- internal/runner/run_task.go
- internal/runner/run_summary.go
- internal/runner/run_tools.go
- internal/runner/run_paths.go
- internal/runner/run_setup.go
- internal/runner/run_types.go
- internal/runner/cucumber.go
- internal/runner/cucumber_feature.go
- internal/runner/cucumber_ground_truth.go
- internal/runner/cucumber_group.go
- internal/runner/cucumber_helpers.go
- internal/runner/cucumber_batch_test.go
- internal/runner/cucumber_errors_test.go
- internal/runner/cucumber_test_helpers_test.go
- internal/tools/runner_types.go
- internal/tools/runner_constructor.go
- internal/tools/runner_list.go
- internal/tools/runner_search.go
- internal/tools/runner_read.go
- internal/tools/runner_paths.go
- internal/tools/runner_exec.go
- internal/tools/runner_output.go
- internal/tools/runner_fs.go
- internal/agent/openrouter.go
- internal/agent/openrouter_messages.go
- internal/agent/openrouter_stream.go
- internal/agent/verbose_constants.go
- internal/agent/verbose_palette.go
- internal/agent/verbose_format.go
- internal/agent/verbose_output.go
- internal/agent/session_types.go
- internal/agent/session_start.go
- internal/agent/session_build.go
- internal/eval/qa_types.go
- internal/eval/qa_eval.go
- internal/eval/qa_schema.go
- internal/eval/qa_citations.go
- internal/eval/qa_deps.go
- internal/config/validate_types.go
- internal/config/validate_collect.go
- internal/config/validate_core.go
- internal/config/validate_agents.go
- internal/config/validate_adapters.go
- internal/config/validate_tasks.go
- internal/config/validate_helpers.go
- internal/cucumber/expectations_types.go
- internal/cucumber/expectations_load.go
- internal/cucumber/expectations_parse.go
- internal/cucumber/expectations_validate.go
- internal/cucumber/godog_types.go
- internal/cucumber/godog_run.go
- internal/cucumber/godog_parse.go
- internal/cucumber/godog_normalize.go
- internal/cucumber/godog_tags.go
- internal/testutil/context.go
- internal/cli/e2e_helpers_test.go (live tag)
- internal/cli/e2e_repo_helpers_test.go (live tag)
- internal/cli/e2e_qa_core_test.go (live tag)
- internal/cli/e2e_qa_repo_test.go (live tag)
- internal/cli/e2e_qa_agents_test.go (live tag)
- internal/cli/e2e_outputs_test.go (live tag)
- internal/cli/e2e_compare_test.go (live tag)
- internal/cli/e2e_init_test.go (live tag)
- internal/cli/e2e_errors_test.go (live tag)
- tests/cucumber/cucumber_test.go (cucumber tag)
- tests/cucumber/steps_state.go (cucumber tag)
- tests/cucumber/steps_config.go (cucumber tag)
- tests/cucumber/steps_run.go (cucumber tag)
- tests/cucumber/steps_assert.go (cucumber tag)
- internal/vcs/git.go
- internal/vcs/git_test.go
- internal/tools/runner_test.go

## Relevant spec documents
- spec/engineering/testing.md
- spec/engineering/repo-structure.md
- spec/engineering/build-and-run.md
- spec/design/api.md
- spec/design/data-model.md
