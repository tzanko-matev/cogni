---
title: Testing
tests:
  - cogni/internal/cli::TestInitCommandCreatesFiles
  - cogni/internal/cli::TestInitCommandRefusesOverwrite
  - cogni/internal/cli::TestValidateCommandSuccess
  - cogni/internal/cli::TestValidateCommandFailure
---

# Testing

## Test strategy

- Unit tests for spec parsing, evaluation, and metrics.
- Integration tests using a small demo repo and fixed prompts.
- End-to-end tests covering full CLI flows and artifact generation.

## Test types

- Unit tests: config parsing, citation validation, metrics aggregation.
- Unit tests: question spec parsing and answer validation.
- Integration tests: live LLM runs against fixture repos.
- E2E tests: CLI workflows (`init`, `validate`, `run`, `compare`, `report`).
  See [spec/roles/working/qa-test-engineer/testing/integration-e2e-tests.md](/working/qa-test-engineer/testing/integration-e2e-tests/) for the suite definition.

## How to run tests

- `go test ./...`

## Docs-linked test results

- Add `tests:` front matter with IDs like `cogni/internal/cli::TestInitCommandCreatesFiles`.
- Run `just docs-test-results` to generate `data/test_results.json`.
- Start the docs site with `just docs-serve` and open this page to see the status panel.
