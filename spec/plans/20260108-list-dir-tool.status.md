# List Dir Tool Status

Status: In progress

ID: 20260108-list-dir-tool.status

Created: 2026-01-08

Linked plan: [spec/plans/20260108-list-dir-tool.plan.md](/plans/20260108-list-dir-tool.plan/)

## Current status
- Phase 2 complete: runner traversal, filesystem helpers, and output formatting implemented.

## What was done so far
- Added `list_dir` tool schema to the tool registry.
- Added `ListDirArgs` type and executor wiring for `list_dir`.
- Implemented BFS traversal, suffix handling, and pagination for list_dir output.
- Extended filesystem abstraction with directory reads and symlink-aware stats.

## Next steps
- Add unit tests for traversal, pagination, and error cases.
- Review output/error message consistency and update docs if needed.

## Latest test run
- 2026-01-08: `nix shell nixpkgs#go -c go test ./internal/tools ./internal/agent`

## Relevant source files (current or planned)
- internal/runner/run_tools.go
- internal/agent/tool_executor.go
- internal/tools/runner_types.go
- internal/tools/runner_list_dir.go
- internal/tools/runner_list_dir_helpers.go
- internal/tools/runner_paths.go
- internal/tools/runner_fs.go
- internal/tools/runner_list.go
- internal/tools/runner_read.go
- internal/tools/runner_output.go
- internal/tools/runner_list_test.go
- internal/tools/runner_test.go

## Relevant spec documents
- spec/inbox/initial-design.md

