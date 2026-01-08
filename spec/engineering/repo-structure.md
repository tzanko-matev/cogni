# Repository Structure

## Top-level layout

- `cmd/cogni`: CLI entrypoint
- `internal/config`: config loading and validation
- `internal/spec`: `.cogni.yml` parsing
- `internal/runner`: task execution pipeline
- `internal/agent`: built-in agent implementation
- `internal/tools`: list_files/list_dir/search/read_file tooling
- `internal/eval`: QA evaluation logic
- `internal/metrics`: metrics collection
- `internal/report`: HTML report generation

## Key directories

- `examples/`: demo repo with sample `.cogni.yml`
- `spec/`: project documentation
