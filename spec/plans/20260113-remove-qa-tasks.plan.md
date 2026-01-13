# Plan: Remove QA Task Logic

Date: 2026-01-13  
Status: TODO

## Goal
Remove all `qa` task logic from code, specs, and documentation so that `question_eval`
is the only supported task type in Cogni.

## Non-goals
- No changes to question_eval semantics or answer parsing.
- No new task types.
- No rate-limiter or concurrency work (separate specs).

## Decisions
- `qa` is not a supported task type anywhere (config, runner, CLI, docs).
- QA-specific config fields and outputs are removed.

## Step 1: Config schema + validation cleanup

Work:
- Remove `qa` from allowed task types.
- Remove QA-only config fields (`prompt`, `eval` block) from `spec.TaskConfig`.
- Update config validation to only accept `question_eval`.
- Update config examples and YAML feature tests to use `question_eval`.

Tests:
- `nix develop -c go test ./internal/config/...` (timeout <= 2s per test)
- `nix develop -c go test ./internal/cli/...` (timeout <= 2s per test)

## Step 2: Runner + evaluation removal

Work:
- Delete `internal/runner/run_task.go` and all `qa` execution paths.
- Remove `internal/eval` package and any QA validation logic.
- Update `runner.Run` dispatch to only `question_eval`.
- Remove QA-only result fields (`Attempts`, `AttemptResult`, `EvalResult`) and
  update JSON outputs and summaries accordingly.

Tests:
- `nix develop -c go test ./internal/runner/...` (timeout <= 2s per test)
- `nix develop -c go test ./internal/cli/...` (timeout <= 2s per test)

## Step 3: CLI flags + behavior

Work:
- Remove the `--repeat` flag (only used for `qa`).
- Update CLI help text and error messages to reflect question_eval-only runs.
- Update CLI tests that reference `qa` to use `question_eval`.

Tests:
- `nix develop -c go test ./internal/cli/...` (timeout <= 2s per test)

## Step 4: Docs + specs cleanup

Work:
- Remove or rewrite any references to `qa` tasks in:
  - `spec/engineering/configuration.md`
  - `spec/design/api.md`
  - `spec/overview/glossary.md`
  - `spec/overview/project-summary.md`
  - `spec/requirements/*` and role guides that mention `qa`
  - `spec/features/*.feature` that use `qa` examples
- Ensure all examples use `question_eval`.

Tests:
- `nix develop -c go test ./...` (sanity check; timeout <= 10s per package)

## Step 5: Repository cleanup

Work:
- Remove now-unused tests and fixtures that only cover `qa`.
- Update any remaining imports or unused code paths.

Tests:
- `nix develop -c go test ./...` (timeout <= 10s per package)

## Done criteria
- No `qa` task type in config validation, code, or docs.
- `question_eval` is the only supported task type.
- All tests updated and passing.
