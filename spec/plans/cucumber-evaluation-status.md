# Cucumber Evaluation Status

Status: In-Progress

## Current status
- Implementation plan expanded with phased work, verification, and exit criteria.
- Phase 0 (Godog dev environment + baseline tests) is defined but not started.
- Go module version aligned to the nix develop Go toolchain (1.25.x).

## What's done so far
- Spec and feature documentation for `cucumber_eval` and adapters.
- Phase 0 requirements captured in the plan.
- Plan aligned with core implementation plan conventions.

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
- 2026-01-06: `nix develop -c env GOMODCACHE=/Users/tzankomatev/.cache/go-mod GOCACHE=/Users/tzankomatev/.cache/go-build go test ./...` failed: `TestDiscoverRepoRootAndMetadata` expected `/tmp/...` but got `/private/tmp/...` in `cogni/internal/vcs`.

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
