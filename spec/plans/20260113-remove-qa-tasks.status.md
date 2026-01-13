# Status: Remove QA Task Logic

ID: 20260113-remove-qa-tasks.status  
Created: 2026-01-13  
Status: NOT STARTED

Linked plan: `spec/plans/20260113-remove-qa-tasks.plan.md`

## Current status
- Plan created. No implementation work started.

## What was done so far
- Defined the scope and steps in the linked plan.

## Next steps
- Step 1: Update config schema/validation to drop `qa` and QA-only fields.
- Step 2: Remove QA execution + evaluation code from runner.
- Step 3: Remove `--repeat` and update CLI tests.
- Step 4: Clean docs/specs referencing `qa`.
- Step 5: Remove unused tests/fixtures and run full test suite.

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
