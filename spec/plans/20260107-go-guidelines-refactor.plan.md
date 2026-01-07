# Go Guidelines Refactor Plan

Status: DONE

ID: 20260107-go-guidelines-refactor

Created: 2026-01-07

Linked status: [spec/plans/20260107-go-guidelines-refactor.status.md](/plans/20260107-go-guidelines-refactor.status/)

## Goal
Bring the Go codebase into compliance with the AGENTS.md refactoring and code
quality guidelines (atomic modularity, functional core/imperative shell,
strict explicitness, docstrings everywhere, deterministic tests with timeouts).

## Scope
- Split oversized Go files into smaller, single-responsibility modules (<200 lines).
- Remove uses of `any` by introducing explicit types or `json.RawMessage`-based models.
- Add docstrings to all packages, types, functions, and methods (exported and unexported).
- Separate pure business logic from I/O and external integrations.
- Replace direct external dependencies in tests (git/rg/LLM/http) with fakes.
- Add per-test timeouts and ensure deterministic execution.
- Ensure all user-visible CLI behaviors are covered by `.feature` + godog tests.
- Provide a single-command live-key test suite (`just test-live`).

## Non-goals
- Changing public CLI behavior or results schema (beyond refactor-only changes).
- Introducing new features or task types.
- Changing evaluation semantics unless needed for correctness or test determinism.

## Inputs and references
- AGENTS.md (refactoring and code quality guidelines)
- spec/engineering/testing.md
- spec/engineering/repo-structure.md
- spec/engineering/build-and-run.md
- spec/design/api.md
- spec/design/data-model.md

## Plan conventions
- Phases are sequential.
- Each phase lists work, verification steps, and exit criteria.
- Build/test gate: run `nix develop -c go test ./...` after code or test changes.

## Phases

### Phase 0 - Inventory and boundaries
- Work:
  - Catalog all Go files >200 lines and identify responsibilities to split.
  - Map I/O boundaries (filesystem, git, rg, HTTP/LLM) and define seams.
  - Identify all `any` usages and decide explicit replacement types.
- Verification:
  - Inventory doc in status file updated with target files and module splits.
- Exit criteria: clear module boundaries and type replacement plan exist.

### Phase 1 - Docstrings and typing cleanup
- Work:
  - Add docstrings to every type/function/method across Go packages.
  - Replace `any` in core data structures with explicit types.
  - For JSON parsing, prefer typed structs or `json.RawMessage` + validators.
- Verification:
  - `rg "\\bany\\b" -g '*.go'` shows no `any` in non-test code.
- Exit criteria: no guideline-violating `any` in core code; docstrings present.

### Phase 2 - Split oversized files and isolate responsibilities
- Work:
  - Split `internal/runner/run.go` into planning, execution, summary, and tooling modules.
  - Split `internal/runner/cucumber.go` into ground-truth loading, prompt rendering,
    agent execution, and result evaluation modules.
  - Split `internal/agent/openrouter.go` into request building, stream parsing,
    and HTTP transport modules.
  - Split `internal/tools/runner.go` into path resolution, command execution,
    and output formatting modules.
- Verification:
  - Each file is <200 lines and adheres to SRP.
- Exit criteria: all oversized files split without behavior changes.

### Phase 3 - Functional core / imperative shell
- Work:
  - Extract pure logic into `core` packages or pure functions.
  - Introduce interfaces for external dependencies (git, rg, filesystem, HTTP).
  - Keep side-effects in thin adapters/wrappers.
- Verification:
  - Core logic functions accept interfaces and are fully unit-testable.
- Exit criteria: core modules have no direct I/O calls.

### Phase 4 - Test determinism and timeouts
- Work:
  - Replace tests that shell out to git/rg/LLM with fakes or fixtures.
  - Add per-test timeouts (context with deadline or `t.Deadline` guard).
  - Move user-visible CLI behaviors into `.feature` + godog where appropriate.
  - Introduce a live-key test suite runnable via `just test-live`.
- Verification:
  - `go test ./...` passes consistently and within 10s in `nix develop`.
- Exit criteria: tests are deterministic, fast, and independent of external tools.

### Phase 5 - Cleanup and verification
- Work:
  - Run gofmt and ensure docs/spec references remain accurate.
  - Remove dead code introduced by splits.
  - Update status file with what changed and latest test runs.
- Verification:
  - `nix develop -c go test ./...` passes.
- Exit criteria: codebase complies with all guidelines.

## Dependencies
- Existing Go toolchain from flake.nix.
- Godog for Cucumber features (already vendored).

## Acceptance criteria
- No Go source file exceeds 200 lines.
- No `any` usage in Go code, including tests.
- All types/functions/methods have docstrings with intent/context.
- Core logic is separated from I/O, with interfaces for external dependencies.
- All tests have explicit timeouts and do not require external binaries or live APIs.
- `nix develop -c go test ./...` passes reliably under 10 seconds.
- `just test-live` runs the live-key suite in one command.

## Completion
- Completed on 2026-01-07.
