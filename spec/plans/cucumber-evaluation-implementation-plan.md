# Cucumber Evaluation Implementation Plan

Status: In-Progress

Linked status: [spec/plans/cucumber-evaluation-status.md](/plans/cucumber-evaluation-status/)

## Goal
Implement `cucumber_eval` tasks that compare an agent's judgment against
Cucumber ground truth, using either the Godog adapter or manual expectations.

## Scope
- Parse feature files and generate stable Example IDs.
- Run Godog with JSON output and normalize results to Example IDs.
- Load manual expectations and map them to Example IDs.
- Score agent answers vs ground truth and write per-example verdicts to
  `results.json`.
- Surface summary accuracy in CLI output and reports.

## Non-goals
- Multi-runner support beyond Godog in MVP.
- Flaky-test detection or retries.
- Custom UI beyond CLI summary and existing report layout.

## Inputs and references
- spec/design/cucumber-evaluation.md
- spec/design/api.md
- spec/design/data-model.md
- spec/engineering/configuration.md
- spec/engineering/testing.md
- spec/engineering/integration-e2e-tests.md
- spec/requirements/functional.md
- spec/requirements/acceptance-criteria.md
- spec/features/cucumber-adapter-godog.feature
- spec/features/cucumber-adapter-manual.feature

## Plan conventions
- Phases are sequential; Phase 0 is required before adapter work.
- Each phase lists work, verification steps, and exit criteria.
- Build/test gate: run `go test ./...` once implementation or tests change.

## Phases

### Phase 0 - Godog dev environment + baseline tests
- Work:
  - Add Godog tooling to the dev environment (pin in `tools.go` and expose via
    `flake.nix`) so `go test ./...` can run feature tests locally.
  - Add a Go test harness (e.g., `tests/cucumber/*_test.go`) that runs Godog
    against a scoped subset of feature files.
  - Tag scenarios that match current functionality (e.g., `@smoke`) and run
    only those in the harness, or explicitly list passing feature files.
  - Implement minimal step definitions for CLI help, config validation, and
    output artifacts that already exist in code.
  - Document how to run the feature tests, including `nix develop` usage.
- Verification:
  - `go test ./...` runs the Godog-based tests in the dev shell.
  - The documented `@smoke` subset passes with current functionality.
- Exit criteria: Godog tests run under `go test` with a known passing baseline.

### Phase 1 - Spec + config support
- Work:
  - Extend config schema for `adapters` and `cucumber_eval` tasks.
  - Validate adapter configuration, feature paths, and expectations directories.
  - Surface Example ID requirements and validation errors clearly.
- Verification:
  - Unit tests for config parsing/validation with invalid adapters and missing files.
  - Sample config loads and validates.
- Exit criteria: configuration supports adapters and Cucumber tasks with clear errors.

### Phase 2 - Feature parsing and Example IDs
- Work:
  - Integrate a Gherkin parser and build a feature index.
  - Implement Example ID generation (tag IDs, scenario IDs, example row IDs).
  - Persist Example ID mapping for use by adapters and evaluation.
- Verification:
  - Unit tests for Example ID generation and parsing across feature fixtures.
- Exit criteria: stable Example IDs are produced for all parsed features.

### Phase 3 - Godog adapter
- Work:
  - Execute Godog with JSON formatter for selected features.
  - Normalize JSON results to Example IDs.
  - Map Godog statuses to implemented/not-implemented ground truth.
- Verification:
  - Integration test that runs Godog on fixture features and produces normalized results.
- Exit criteria: Godog adapter produces Example ID keyed results.

### Phase 4 - Manual expectations adapter
- Work:
  - Define expectations file format and loader.
  - Map expectations to Example IDs and validate coverage.
  - Report missing or duplicate Example IDs.
- Verification:
  - Unit tests for loader and validation with fixture expectations.
- Exit criteria: manual expectations produce complete ground truth or clear errors.

### Phase 5 - Evaluation + outputs
- Work:
  - Compare agent decisions to ground truth per example.
  - Record per-example verdicts, accuracy, and evidence in `results.json`.
  - Emit CLI summary for `cucumber_eval` tasks.
- Verification:
  - Integration tests covering both adapters and output shapes.
- Exit criteria: results include per-example verdicts and accuracy.

### Phase 6 - End-to-end testing
- Work:
  - E2E tests for `cogni run`, `cogni report`, and `cogni compare` with
    Cucumber tasks.
  - Ensure outputs remain stable for CI usage.
- Verification:
  - E2E run against fixture repo produces expected JSON and summary output.
- Exit criteria: E2E tests pass with deterministic outputs.

## Dependencies
- Gherkin parser library for Go.
- Godog binary or module available in dev environment for tests.
- JSON formatter support in Godog used by the adapter.

## Acceptance criteria
- `cogni run` supports `cucumber_eval` tasks with Godog or manual expectations.
- Per-example verdicts and accuracy appear in `results.json`.
- CLI summary reports per-example accuracy for Cucumber tasks.
- Missing feature files, expectations, or Godog failures produce clear errors.

## Risks
- Example ID stability if tags are not enforced.
- Inconsistent Godog JSON output across versions.
- Feature parsing mismatches with test runner discovery.
