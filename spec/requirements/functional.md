# Functional Requirements

## Core capabilities

- Initialize `.cogni.yml` and schema folder (`cogni init`).
- Validate YAML and referenced schemas (`cogni validate`).
- Run the full benchmark (`cogni run`).
- Run a subset of tasks using selectors (`task-id` or `task-id@agent-id`).
- Configure multiple agents and select per task; support `--agent` override.
- Support `cogni run --verbose` to stream LLM input/output, tool calls and results, and per-task metrics to the console (respect truncation limits).
- Capture metrics per attempt: correctness, tokens, wall time, tool calls, files read, model, agent ID.
- Write outputs to `<output_dir>/<commit>/<run-id>/` (`results.json`, `report.html`, logs).
- Compare runs by base/head or commit range (`cogni compare`).
- Generate reports with trend charts over a commit range (`cogni report`).

## Edge cases

- Unknown task ID or agent ID must fail with a clear error.
- Invalid YAML or JSON schema must fail validation before running.
- Missing `LLM_API_KEY` must fail fast.
- Commit ranges with missing runs must warn and continue.
- Citation validation failures must mark a task as failed.
- Budget overruns must mark a task as failed with `budget_exceeded`.
