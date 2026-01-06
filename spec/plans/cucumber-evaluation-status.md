# Cucumber Evaluation Status

Status: Planned

## Current status
- Spec and feature documentation exists for `cucumber_eval` and adapters.
- No implementation work has started in code.

## What's done so far
- Design and requirements updates across spec docs.
- Feature files added to describe Godog and manual adapters.

## Next steps
- Define config structs and validation for `adapters` and `cucumber_eval` tasks.
- Implement Example ID generation and feature parsing.
- Build Godog adapter runner + JSON normalizer.
- Build manual expectations loader and matcher.
- Add per-example verdicts to `results.json` and CLI summaries.
- Add tests (unit + integration + e2e) for adapters and evaluation flow.

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
