# Cucumber Evaluation Status

Status: In-Progress

## Current status
- Phase 0 completed with Godog tooling, smoke tags, and a Cucumber test harness.
- Phase 1 completed: config structs and validation for adapters and cucumber_eval tasks.
- Phase 2 completed: Gherkin parsing and Example ID generation with unit tests.
- Phase 3 in progress: Godog JSON runner and normalization to Example IDs.
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
- Normalized repo root expectations in vcs tests to handle `/tmp` vs `/private/tmp`.

## Next steps
- Install Godog in the dev environment and wire feature tests into `go test`.
- Add Go test harnesses for a subset of feature files, with at least some passing scenarios.
- Extend config structs and validation for `adapters` and `cucumber_eval` tasks.
- Implement Example ID generation and feature parsing.
- Build Godog adapter runner and JSON normalizer.
- Build manual expectations loader and matcher.
- Add per-example verdicts to `results.json` and CLI summaries.
- Add tests (unit, integration, E2E) for adapters and evaluation flow.

## Latest test run
- 2026-01-06: `nix develop -c env LLM_API_KEY= OPENROUTER_API_KEY= GOPATH=/Users/tzankomatev/.cache/go GOMODCACHE=/Users/tzankomatev/.cache/go-mod GOCACHE=/Users/tzankomatev/.cache/go-build go test ./...` passed.

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
