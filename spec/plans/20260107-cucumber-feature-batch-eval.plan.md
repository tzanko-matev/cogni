# Cucumber Feature-Batch Evaluation Plan

Status: Planned

ID: 20260107-cucumber-feature-batch-eval

Created: 2026-01-07

Linked status: [spec/plans/20260107-cucumber-feature-batch-eval.status.md](/plans/20260107-cucumber-feature-batch-eval.status/)

## Goal
Replace per-example agent runs with one agent run per feature file for
`cucumber_eval` tasks. Each agent run may involve multiple LLM turns, but the
unit of evaluation is the feature file (not individual scenarios). Keep
correctness per example and add feature-level effort metrics. Missing or extra
example results must be treated as errors.

## Scope
- Remove per-example agent prompting and response parsing.
- Introduce batch (per-feature) agent response schema and strict validation.
- Include feature path and full feature text in the LLM prompt.
- Record feature-level effort metrics in `results.json`.
- Keep per-example correctness, ground truth, and verdicts unchanged.
- Update specs, config docs, and feature files to reflect the new behavior.
- Update unit, integration, and E2E tests.

## Non-goals
- Changing adapter behavior (Godog/manual ground truth stays as-is).
- Altering Example ID generation rules.
- Adding new task types or config modes.
- Aggregating multiple feature files into a single LLM call.

## Inputs and references
- spec/design/cucumber-evaluation.md
- spec/design/api.md
- spec/design/data-model.md
- spec/engineering/configuration.md
- spec/requirements/functional.md
- spec/requirements/acceptance-criteria.md
- spec/features/cucumber-adapter-godog.feature
- spec/features/cucumber-adapter-manual.feature
- internal/runner/cucumber.go
- internal/cucumber/agent.go
- internal/runner/results.go

## Plan conventions
- Phases are sequential.
- Each phase lists work, verification steps, and exit criteria.
- Build/test gate: run `go test ./...` after code or tests change.

## Phases

### Phase 0 - Specs and documentation
- Work:
  - Update cucumber evaluation design to specify per-feature LLM calls.
  - Define batch response JSON schema and strict missing/extra validation.
  - Document prompt template placeholders for feature path/text and expected IDs.
  - Update API/config docs and examples to match the new prompt/response shape.
  - Update feature specs to describe per-feature evaluation behavior.
- Verification:
  - Spec docs no longer describe per-example prompts or responses.
- Exit criteria: specs and examples reflect per-feature evaluation only.

### Phase 1 - Results schema and data model
- Work:
  - Add feature-level effort metrics to `results.json` (e.g., tokens/time/tools).
  - Introduce a `feature_runs` (or similar) section under `CucumberEval`.
  - Ensure summaries remain per-example for correctness and accuracy.
- Verification:
  - Unit tests updated or added for JSON serialization of new fields.
- Exit criteria: results model captures feature-level effort metrics.

### Phase 2 - Batch agent response parsing
- Work:
  - Replace single-result parsing with batch response parsing.
  - Validate every result has an `example_id`, all IDs are unique, and the set
    matches expected Example IDs for the feature (missing/extra -> error).
  - Preserve evidence/notes parsing per example.
- Verification:
  - Unit tests for valid batch responses, missing IDs, extra IDs, and duplicates.
- Exit criteria: batch responses are parsed and validated deterministically.

### Phase 3 - Runner implementation (per-feature calls)
- Work:
  - Group examples by feature file and load feature text.
- Render a single prompt per feature (include path + contents + expected IDs).
  - Run the agent once per feature file (allowing multiple turns) and parse the
    final batch response.
  - Map results to per-example verdicts and compute correctness.
  - Record feature-level effort metrics for each feature run.
  - Remove per-example prompt rendering and per-example LLM calls.
- Verification:
  - Unit tests for `runCucumberTask` cover per-feature execution and errors.
- Exit criteria: no per-example LLM calls remain; per-feature flow works end-to-end.

### Phase 4 - Test updates and E2E coverage
- Work:
  - Update cucumber runner tests to use batch responses.
  - Update E2E tests and fixtures for the new prompt/response format.
  - Update Cucumber feature tests for the new behavior.
- Verification:
  - `go test ./...` passes in `nix develop`.
- Exit criteria: tests pass with per-feature evaluation only.

### Phase 5 - Cleanup
- Work:
  - Remove obsolete structs/helpers and unused fields tied to per-example calls.
  - Ensure docs and examples no longer mention the old mode.
- Verification:
  - `rg` shows no remaining per-example prompt flow references.
- Exit criteria: codebase and docs are consistent with per-feature evaluation.

## Dependencies
- Gherkin parser (existing).
- Godog (existing) for ground truth when using the cucumber adapter.

## Acceptance criteria
- `cogni run` performs exactly one LLM call per feature file.
- Missing or extra `example_id` results cause task failure with clear errors.
- `results.json` includes feature-level effort metrics and per-example correctness.
- CLI summary remains per-example accuracy for `cucumber_eval` tasks.
