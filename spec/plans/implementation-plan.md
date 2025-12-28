# Implementation Plan

## Scope (MVP)

- Single Go CLI binary: `cogni` with `init`, `validate`, `run`, `compare`, `report`.
- QA-only tasks with objective evaluation (JSON/schema/citations).
- Built-in agent only, OpenRouter provider only, git-only repo integration.
- Local, read-only execution with outputs under `<output_dir>/<commit>/<run-id>/`.

## Inputs and references

- Requirements: `spec/requirements/functional.md`, `spec/requirements/non-functional.md`.
- Architecture: `spec/architecture/logical-architecture.md`, `spec/architecture/data-flow.md`.
- Design: `spec/design/api.md`, `spec/design/data-model.md`, `spec/design/ui-ux.md`.
- Engineering: `spec/engineering/repo-structure.md`, `spec/engineering/configuration.md`,
  `spec/engineering/testing.md`, `spec/engineering/build-and-run.md`,
  `spec/engineering/observability.md`, `spec/engineering/ci-cd.md`.
- Built-in agent behavior: `spec/engineering/builtin-agent.md`.
- Acceptance criteria: `spec/requirements/acceptance-criteria.md`.

## Plan conventions

- Phases are sequential; map to milestones as: M1 (Phases 0-1), M2 (Phases 2-5),
  M3 (Phases 6-7), M4 (Phase 8).
- Each phase lists key work items, verification steps, and exit criteria.
- Build/test gates: use `go build ./cmd/cogni` and `go test ./...` once code exists.

## Cross-cutting requirements (MVP)

- Read-only tools; no repository writes.
- Deterministic task order and stable output paths.
- Use `rg` for search and enforce file/tool output limits.
- Enforce budgets (tokens, steps, wall time) with `budget_exceeded` failures.
- Local-only outputs under `<output_dir>/<commit>/<run-id>/`.

## Phase 0 - Repository and CLI scaffolding

- Inputs: `spec/engineering/repo-structure.md`, `spec/design/api.md`,
  `spec/engineering/build-and-run.md`.
- Work:
  - Initialize Go module and directory layout (`cmd/cogni`, `internal/*`).
  - Create CLI entrypoint with subcommands and help text; define exit codes.
  - Add shared config/result structs for `.cogni.yml` and output artifacts.
- Verification:
  - `go build ./cmd/cogni`.
  - `cogni --help` lists all commands with usage text.
- Deliverable: `cogni` builds and prints help for all commands.

## Phase 1 - Spec parsing and validation

- Inputs: `spec/engineering/configuration.md`, `spec/design/data-model.md`,
  `spec/requirements/functional.md`.
- Work:
  - Load `.cogni.yml` into config structs with defaults and normalization.
  - Validate unique task/agent IDs, default agent references, budgets, and
    schema file paths.
  - Validate referenced JSON schemas are syntactically valid and loadable.
  - Implement `cogni init` to scaffold `.cogni.yml` plus `schemas/` examples.
  - Implement `cogni validate` with actionable errors (file, field, reason).
- Verification:
  - Unit tests for parsing defaults, invalid YAML, duplicate IDs, bad schemas.
  - CLI tests for `cogni init` output and `cogni validate` error text.
- Deliverable: sample config validates; invalid config fails fast.

## Phase 2 - VCS and workspace handling

- Inputs: `spec/architecture/system-context.md`, `spec/architecture/data-flow.md`,
  `spec/requirements/functional.md`.
- Work:
  - Detect git repo root; capture commit SHA, branch, and dirty state metadata.
  - Resolve refs for `compare`/`report` (base/head and range syntax).
  - Define run ID generation and output directory layout conventions.
  - Ensure deterministic task ordering (stable by config order or ID).
- Verification:
  - Unit tests for ref/range parsing and run ID formatting.
  - Fixture-based tests that resolve commit ranges and output paths.
- Deliverable: stable commit metadata and output paths.

## Phase 3 - Tool layer and instrumentation

- Inputs: `spec/requirements/non-functional.md`, `spec/engineering/observability.md`,
  `spec/engineering/builtin-agent.md`.
- Work:
  - Implement read-only tools: `list_files`, `search` (rg), `read_file`.
  - Enforce file read limits, output truncation, and error surfaces.
  - Record tool call timings, errors, and output sizes for metrics.
  - Define structured tool outputs for downstream evaluation and logging.
- Verification:
  - Unit tests for file limits, truncation behavior, and error cases.
  - Integration tests that exercise `rg` search and file reads on fixtures.
- Deliverable: tool registry returns structured outputs with metrics hooks.

## Phase 4 - Built-in agent runtime

- Implement session init, prompt building, and tool loop per `spec/engineering/builtin-agent.md`.
- Enforce budgets (steps, time, tokens) and capture token usage if available.
- Support per-task agent selection and `--agent` override.
- Deliverable: a QA task runs end-to-end with tool usage and final JSON output.

## Phase 5 - Evaluation and metrics

- Implement QA evaluation pipeline:
  - JSON parse
  - JSON schema validation
  - must_contain checks
  - citation validation
- Define failure reasons and persist evaluation artifacts.
- Aggregate per-attempt metrics and run summary.
- Deliverable: tasks report pass/fail with concrete failure reasons.

## Phase 6 - Runner pipeline

- Orchestrate tasks: selection, execution, retries (`--repeat`), and aggregation.
- Write `results.json`, `report.html`, and per-task logs to output dir.
- Ensure results are written even when some tasks fail.
- Deliverable: `cogni run` produces stable outputs and CLI summary.

## Phase 7 - Reporting and compare

- Implement results loader and comparison logic (base/head or range).
- Generate `report.html` with summary, task table, and trend charts.
- Implement `cogni report` and `cogni compare` outputs and error handling.
- Deliverable: compare/report works over commit ranges with warnings for missing runs.

## Phase 8 - Docs, examples, and CI

- Provide example configs and sample question suite under `examples/`.
- Update docs for setup/build/run, troubleshooting, and CI workflow.
- Add a CI smoke test with a fixture repo and golden report.
- Deliverable: a new repo can run a benchmark in under 15 minutes.

## Testing plan (by phase)

- Unit tests: config parsing, schema validation, citation checks, metrics aggregation.
- Integration tests: end-to-end run on a fixture repo producing `results.json`.
- CLI tests: flag parsing, error messages, and range resolution.

## Definition of done (MVP)

- All acceptance criteria in `spec/requirements/acceptance-criteria.md` pass.
- `go test ./...` passes locally.
- `cogni run`, `cogni compare`, and `cogni report` work on a demo repo.

## Post-MVP follow-ups

- Multi-provider and external agent adapters.
- Multi-VCS support (jujutsu and others).
- Sandboxed runners and hosted report sharing.
