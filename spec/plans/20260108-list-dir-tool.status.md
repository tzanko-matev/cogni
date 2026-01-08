# List Dir Tool Status

Status: In progress

ID: 20260108-list-dir-tool.status

Created: 2026-01-08

Linked plan: [spec/plans/20260108-list-dir-tool.plan.md](/plans/20260108-list-dir-tool.plan/)

## Current status
- Not started. Plan drafted and awaiting spec confirmations.

## What was done so far
- None.

## Next steps
- Confirm sorting/pagination output rules with the spec owner.
- Add tool schema and executor wiring.
- Implement runner traversal and filesystem helpers.
- Add unit tests for traversal, pagination, and errors.

## Latest test run
- Not run yet.

## Relevant source files (current or planned)
- internal/runner/run_tools.go
- internal/agent/tool_executor.go
- internal/tools/runner_types.go
- internal/tools/runner_paths.go
- internal/tools/runner_fs.go
- internal/tools/runner_list.go
- internal/tools/runner_read.go
- internal/tools/runner_output.go
- internal/tools/runner_list_test.go
- internal/tools/runner_test.go

## Relevant spec documents
- spec/inbox/initial-design.md
