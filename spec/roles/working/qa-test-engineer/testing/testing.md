# Testing

## Test strategy

- Unit tests for spec parsing, evaluation, and metrics.
- Integration tests using a small demo repo and fixed prompts.
- End-to-end tests covering full CLI flows and artifact generation.

## Test types

- Unit tests: config parsing, citation validation, metrics aggregation.
- Integration tests: live LLM runs against fixture repos.
- E2E tests: CLI workflows (`init`, `validate`, `run`, `compare`, `report`).
  See `spec/engineering/integration-e2e-tests.md` for the suite definition.

## How to run tests

- `go test ./...`
