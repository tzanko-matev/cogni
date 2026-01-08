# Logical Architecture

## Major components

- CLI entrypoint (command parsing and dispatch)
- Spec parser and validator (`.cogni/config.yml` + `.cogni/schemas/`, discovered via parent search)
- VCS resolver (commit and range handling; git in MVP)
- Agent manager (agent selection and configuration)
- Tool layer (list_files/list_dir/search/read_file)
- Evaluator (JSON parsing, schema validation, citation checks)
- Metrics collector (tokens, time, tool calls, files read)
- Results writer (`results.json` + logs)
- Report generator (`report.html`)

## Responsibilities

- CLI routes user intent to the correct workflow.
- Spec parser loads tasks, agents, and budgets.
- VCS resolver expands commit ranges for `compare` and `report`.
- Agent manager runs tasks with the configured agent.
- Evaluator decides pass/fail based on objective checks.
- Results writer persists outputs to the configured `output_dir`.

## Key interactions

- `run`: parse spec -> resolve agent -> execute tools -> evaluate -> write results -> render report.
- `compare`: resolve base/head or range -> load results -> compute deltas -> print summary.
- `report`: resolve range -> load results -> render charts and tables.
