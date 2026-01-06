# Cucumber Evaluation Status

Status: In-Progress

## Current status
- Phase 0 completed with Godog tooling, smoke tags, and a Cucumber test harness.
- Phase 1 completed: config structs and validation for adapters and cucumber_eval tasks.
- Phase 2 completed: Gherkin parsing and Example ID generation with unit tests.
- Phase 3 completed: Godog JSON runner and normalization to Example IDs.
- Phase 4 completed: manual expectations loader and validation.
- Phase 5 completed: cucumber_eval execution, results output, and CLI summaries.
- Phase 6 in progress: end-to-end CLI coverage for cucumber_eval tasks.
- Go module version aligned to the nix develop Go toolchain (1.25.x).
- Dev shell sets Go caches and installs Godog automatically if missing.

## What's done so far
- Spec and feature documentation for `cucumber_eval` and adapters.
- Phase 0 guidance captured in the plan, including smoke subset guidance.
- Added Godog tooling pin (`tools/tools.go`) and dev-shell install hook.
- Added Cucumber smoke harness + step definitions for CLI help and validation.
- Tagged CLI feature smoke scenarios for baseline Godog runs.
- Extended config types for adapters and cucumber_eval task fields.
- Added validation for adapters, features, and prompt_template requirements.
- Added Cucumber feature parser with Example ID generation and unit tests.
- Added Godog JSON parser, scenario status normalization, and example indexing utilities.
- Added manual expectations loader/validator with tests.
- Added cucumber_eval task execution with per-example results and summaries.
- Added CLI summary line for Cucumber task accuracy.
- Normalized repo root expectations in vcs tests to handle `/tmp` vs `/private/tmp`.

## Next steps
- Add end-to-end CLI coverage for cucumber_eval tasks (`run`, `compare`, `report`).
- Confirm results output includes per-example verdicts and summary fields in real runs.

## Latest test run
- 2026-01-06: `nix develop -c env LLM_API_KEY= OPENROUTER_API_KEY= GOPATH=/Users/tzankomatev/.cache/go GOMODCACHE=/Users/tzankomatev/.cache/go-mod GOCACHE=/Users/tzankomatev/.cache/go-build go test ./...` passed (Phase 5 updates).

## Relevant source files (current or planned)
- internal/config/* (config structs, validation)
- internal/spec/* (parsing, schema handling)
- internal/runner/* (task orchestration, results output)
- internal/eval/* (new cucumber_eval evaluation)
- internal/cli/* (run summary updates)
- internal/report/* (report additions for per-example accuracy)

## Relevant spec documents
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
