# List Dir Tool Plan

Status: In progress

ID: 20260108-list-dir-tool

Created: 2026-01-08

Linked status: [spec/plans/20260108-list-dir-tool.status.md](/plans/20260108-list-dir-tool.status/)

## Goal
Add a new read-only `list_dir` tool that mirrors Codex-style directory listings
with depth limits and pagination while honoring repo root safety rules.

## Scope
- Define tool schema/registry entry and executor dispatch.
- Implement BFS directory traversal with sorting, depth limits, pagination, and
  Codex-style formatting/suffixes.
- Enforce path validation and repo-root safety using the existing resolver.
- Add unit tests for traversal, formatting, pagination, and error cases.

## Non-goals
- Globbing or filtering by pattern.
- Hidden-file filtering (include hidden entries by default).
- Any write capability or repo mutations.

## Inputs and references
- spec/inbox/initial-design.md (tool inventory conventions)
- internal/runner/run_tools.go
- internal/agent/tool_executor.go
- internal/tools/runner_paths.go
- internal/tools/runner_read.go
- internal/tools/runner_list.go
- internal/tools/runner_output.go
- internal/tools/runner_fs.go
- internal/tools/runner_types.go

## Plan conventions
- Phases are sequential.
- Build/test gate: run `go test ./internal/tools ./internal/agent` after code
  or tests change.

## Phases

### Phase 0 - Spec confirmation
- Confirm sorting rules around pagination (global BFS + re-sort slice) and the
  wording of the "More than {limit} entries found" line.
- Decide whether output should list full relative paths or entry names with
  indentation (example implies names).

### Phase 1 - Tool registration and args
- Add `list_dir` schema in tool registry with required `path` and optional
  `offset`, `limit`, `depth` fields.
- Add `ListDirArgs` to tool runner types.
- Add executor handling for `list_dir` args and defaults.

### Phase 2 - Runner implementation
- Extend the filesystem abstraction to support directory reads and symlink
  detection.
- Implement BFS traversal with depth limits, stable sorting, and suffix rules.
- Normalize output separators to `/`, truncate names to 500 bytes safely, and
  format indentation.
- Apply pagination and emit the trailing "More than ..." line as specified.

### Phase 3 - Tests
- Cover depth 1/2/3 traversal and indentation.
- Verify symlink suffix and non-traversal behavior.
- Validate pagination, offset bounds, and the "More than" marker.
- Confirm errors for invalid path, escaping, non-dir, and invalid params.

### Phase 4 - Cleanup & documentation
- Ensure consistent error messages and output formatting.
- Update any internal docs if needed (tool list references).

## Acceptance criteria
- `list_dir` is available in tool definitions and executed by the runner.
- Output format matches the spec (prefix line + formatted entries).
- All unit tests covering traversal, pagination, and errors pass.
