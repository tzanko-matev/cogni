# Status: Remove QA Task Logic

ID: 20260113-remove-qa-tasks.status  
Created: 2026-01-13  
Status: IN PROGRESS

Linked plan: `spec/plans/20260113-remove-qa-tasks.plan.md`

## Current status
- Step 1-3 in progress: QA task types removed from config/runner/CLI, tests updated, eval package removed.

## What was done so far
- Removed `qa` fields from task schema and validation; only `question_eval` is accepted.
- Removed QA runner path and `internal/eval` package.
- Removed `--repeat` flag and RunParams repeat support.
- Updated runner results to only include question_eval outputs.
- Updated config scaffolding and unit tests to question_eval.
- Updated live e2e CLI tests to question_eval semantics.
- Verified `nix develop -c go test ./internal/...` passes (2026-01-13).

## Next steps
- Step 4: Clean docs/specs referencing `qa`.
- Step 5: Remove any remaining QA references in specs/features and run full test suite.

## Relevant source files (current or planned)
- `internal/spec/types.go`
- `internal/config/validate_tasks.go`
- `internal/runner/run.go`
- `internal/runner/run_task.go`
- `internal/runner/results.go`
- `internal/runner/run_summary.go`
- `internal/eval/*`
- `internal/cli/run.go`
- `internal/cli/run_test.go`
- `internal/cli/validate_test.go`
- `internal/cli/e2e_qa_*`
- `internal/runner/run_test.go`

## Relevant spec documents
- `spec/design/question-evaluation.md`
- `spec/engineering/configuration.md`
- `spec/design/api.md`
- `spec/overview/glossary.md`
- `spec/requirements/*`
