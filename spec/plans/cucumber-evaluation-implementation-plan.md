# Cucumber Evaluation Implementation Plan

Status: Planned

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

## Phases

### Phase 1 - Spec + config support
- Extend config schema for `adapters` and `cucumber_eval` tasks.
- Validate adapter configuration, feature paths, and expectations directory.
- Add Example ID rules to config validation error messages.

### Phase 2 - Feature parsing and Example IDs
- Add Gherkin parser integration for feature discovery.
- Implement Example ID generation with tag-based IDs and example row IDs.
- Provide a fallback ID only when explicit IDs are missing.

### Phase 3 - Godog adapter
- Execute Godog with JSON formatter for selected features.
- Normalize JSON results to Example IDs.
- Map Godog statuses to implemented/not implemented.

### Phase 4 - Manual expectations adapter
- Define expectations file format and loader.
- Map expectations to Example IDs.
- Validate expectation coverage and report missing Example IDs.

### Phase 5 - Evaluation + outputs
- Compare agent decisions to ground truth per example.
- Record per-example verdicts, accuracy, and evidence in `results.json`.
- Emit CLI summary for `cucumber_eval` tasks.

### Phase 6 - Testing
- Unit tests for Example ID generation and expectations parsing.
- Integration tests for Godog and manual adapters using fixture features.
- E2E tests for CLI flows (run, results, report).

## Dependencies
- Gherkin parser library for Go.
- Godog binary available in PATH for integration tests.

## Acceptance criteria
- `cogni run` supports `cucumber_eval` tasks with Godog or manual expectations.
- Per-example verdicts and accuracy appear in `results.json`.
- CLI summary reports per-example accuracy for Cucumber tasks.
- Missing feature files, expectations, or Godog failures produce clear errors.

## Risks
- Example ID stability if tags are not enforced.
- Inconsistent Godog JSON output across versions.
- Feature parsing mismatches with test runner discovery.
