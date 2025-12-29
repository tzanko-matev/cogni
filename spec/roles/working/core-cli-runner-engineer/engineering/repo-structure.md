# Repository Structure

## Top-level layout

- `cmd/cogni`: CLI entrypoint
- `internal/config`: config loading and validation
- `internal/spec`: `.cogni/config.yml` parsing and `.cogni/` discovery
- `internal/runner`: task execution pipeline
- `internal/agent`: built-in agent implementation
- `internal/tools`: list/search/read tooling
- `internal/eval`: QA evaluation logic
- `internal/metrics`: metrics collection
- `internal/report`: HTML report generation

## Key directories

- `examples/`: demo repo with sample `.cogni/config.yml`
- `spec/`: project documentation
