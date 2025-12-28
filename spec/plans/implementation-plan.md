# Implementation Plan

## Scope (MVP)

- Single Go CLI binary: `cogni` with `init`, `validate`, `run`, `compare`, `report`.
- QA-only tasks with objective evaluation (JSON/schema/citations).
- Built-in agent only, OpenRouter provider only, git-only repo integration.
- Local, read-only execution with outputs under `<output_dir>/<commit>/<run-id>/`.

## Inputs and references

- Requirements: `spec/requirements/functional.md`, `spec/requirements/non-functional.md`.
- Architecture: `spec/architecture/logical-architecture.md`, `spec/architecture/data-flow.md`.
- Built-in agent behavior: `spec/engineering/builtin-agent.md`.
- Acceptance criteria: `spec/requirements/acceptance-criteria.md`.

## Phase 0 - Repository and CLI scaffolding

- Create Go module layout per `spec/engineering/repo-structure.md`.
- Add CLI skeleton with subcommands and flags; wire help text and exit codes.
- Define shared config structs for `.cogni.yml` and results output.
- Deliverable: `cogni` builds and prints help for all commands.

## Phase 1 - Spec parsing and validation

- Implement YAML loading for `.cogni.yml` with defaults and validation.
- Validate task IDs, agent IDs, budgets, and referenced JSON schemas.
- Implement `cogni init` to scaffold a starter `.cogni.yml` and `schemas/`.
- Implement `cogni validate` with clear, actionable error messages.
- Deliverable: sample config validates; invalid config fails fast.

## Phase 2 - VCS and workspace handling

- Detect git repo root; resolve current commit and branch.
- Implement ref and range resolution for `compare` and `report`.
- Define run ID generation and output directory layout.
- Ensure deterministic task ordering for a run.
- Deliverable: stable commit metadata and output paths.

## Phase 3 - Tool layer and instrumentation

- Implement read-only tools: `list_files`, `search` (rg), `read_file`.
- Enforce output size limits and file read bounds.
- Log tool calls and timings for metrics aggregation.
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
