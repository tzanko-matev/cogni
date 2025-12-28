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

- Inputs: `spec/engineering/builtin-agent.md`, `spec/engineering/configuration.md`,
  `spec/requirements/functional.md`.
- Work:
  - Implement session initialization, prompt building, and tool loop as specified.
  - Integrate OpenRouter request/stream handling for the built-in agent.
  - Fail fast when `LLM_API_KEY` is missing or provider config is unsupported.
  - Enforce budgets (steps, time, tokens) and surface `budget_exceeded`.
  - Support per-task agent selection and `--agent` override.
  - Capture token usage, tool counts, and wall time metrics.
- Verification:
  - Unit tests for initial context, prompt building, and compaction rules.
  - Fake provider tests for tool-call loops and streaming sequences.
  - Preflight tests for missing API keys and provider selection errors.
  - Integration test that runs a QA task and produces JSON output.
- Deliverable: a QA task runs end-to-end with tool usage and final JSON output.

## Phase 5 - Evaluation and metrics

- Inputs: `spec/design/data-model.md`, `spec/requirements/functional.md`,
  `spec/engineering/testing.md`.
- Work:
  - Implement QA evaluation pipeline: JSON parse, schema validation,
    must_contain checks, and citation validation.
  - Define failure reasons and persist evaluation artifacts per attempt.
  - Aggregate per-attempt metrics and run summaries for `results.json`.
- Verification:
  - Unit tests for each evaluation step and failure reason mapping.
  - Fixture tests that compare evaluation outputs to expected results.
- Deliverable: tasks report pass/fail with concrete failure reasons.

## Phase 6 - Runner pipeline

- Inputs: `spec/architecture/logical-architecture.md`, `spec/design/data-model.md`,
  `spec/engineering/observability.md`, `spec/requirements/functional.md`.
- Work:
  - Orchestrate task selection, execution, repeats (`--repeat`), and aggregation.
  - Validate task selectors (`task-id`, `task-id@agent-id`) and fail on unknown IDs.
  - Run repo setup commands from config (if provided) before task execution.
  - Define `results.json` schema fields (VCS metadata, agent/model, per-attempt metrics).
  - Write `results.json`, `report.html`, and per-task logs to output dir.
  - Ensure results are written even when some tasks fail.
  - Emit CLI summary output per run.
- Verification:
  - Integration test on a fixture repo with partial failures and repeats.
  - CLI tests for invalid task/agent selectors and missing defaults.
  - Assert outputs are written with stable paths and summary fields.
- Deliverable: `cogni run` produces stable outputs and CLI summary.

## Phase 7 - Reporting and compare

- Inputs: `spec/design/ui-ux.md`, `spec/design/api.md`,
  `spec/requirements/functional.md`.
- Work:
  - Implement results loader and comparison logic (base/head or range).
  - Load reports from stored run artifacts; avoid rerunning tasks.
  - Generate `report.html` with summary, task table, and trend charts.
  - Implement `cogni report` and `cogni compare` output and error handling.
  - Support `--open` to launch the rendered report when available.
  - Warn on missing runs in ranges while rendering remaining data.
- Verification:
  - Golden tests for report HTML and summary outputs.
  - Fixture tests for compare/report with missing runs and invalid ranges.
- Deliverable: compare/report works over commit ranges with warnings for missing runs.

## Phase 8 - Docs, examples, and CI

- Inputs: `spec/engineering/setup.md`, `spec/engineering/ci-cd.md`,
  `spec/engineering/build-and-run.md`, `spec/engineering/troubleshooting.md`,
  `spec/engineering/deployment.md`.
- Work:
  - Provide example configs and sample question suite under `examples/`.
  - Update docs for setup/build/run, troubleshooting, and CI workflow.
  - Add a CI smoke test with a fixture repo and a golden report.
  - Document release steps for local/CI distribution.
- Verification:
  - CI runs `go test ./...` and `go build ./cmd/cogni` plus the smoke test.
  - Documentation walkthrough completes a benchmark in under 15 minutes.
- Deliverable: a new repo can run a benchmark in under 15 minutes.

## Testing and quality gates

- Unit tests: config parsing, tool limits, prompt building, eval checks, metrics.
- Integration tests: fixture repo run producing `results.json` and `report.html`.
- CLI tests: help text, exit codes, error messages, and range resolution.
- Golden tests: report HTML and summary outputs.

## Acceptance criteria traceability

- `.cogni.yml` support and validation: Phases 1 and 6.
- `cogni validate` behavior: Phase 1.
- `cogni run` outputs (`results.json`, `report.html`): Phase 6.
- `cogni compare` and `cogni report`: Phase 7.
- QA failure cases (JSON/schema/citations/budget): Phases 3-5.

## Definition of done (MVP)

- All acceptance criteria in `spec/requirements/acceptance-criteria.md` pass.
- `go test ./...` and `go build ./cmd/cogni` pass locally.
- `cogni run`, `cogni compare`, and `cogni report` work on a demo repo.
- CI smoke test passes with `results.json` and `report.html` artifacts.

## Post-MVP follow-ups

- Multi-provider and external agent adapters.
- Multi-VCS support (jujutsu and others).
- Sandboxed runners and hosted report sharing.
