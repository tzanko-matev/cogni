# Functional Requirements

## Core capabilities

- Initialize `.cogni/` with `.cogni/config.yml` and `.cogni/schemas/` after user confirmation (`cogni init`).
- If a git repo is detected, `cogni init` suggests the repo root; otherwise it targets the current directory.
- Prompt for a results folder during `cogni init` (default `.cogni/results`) and write it to `repo.output_dir`.
- If a git repo is detected, offer to add the results folder to the repo root `.gitignore`.
- Validate YAML and referenced schemas (`cogni validate`).
- Run the full benchmark (`cogni run`).
- Run a subset of tasks using selectors (`task-id` or `task-id@agent-id`).
- Configure multiple agents and select per task; support `--agent` override.
- Capture metrics per attempt: correctness, tokens, wall time, tool calls, files read, model, agent ID.
- Write outputs to `<output_dir>/<commit>/<run-id>/` (`results.json`, `report.html`, logs).
- Compare runs by base/head or commit range (`cogni compare`).
- Generate reports with trend charts over a commit range (`cogni report`).
- Resolve `.cogni/` by searching parent directories when commands run outside the config folder.

## Edge cases

- Unknown task ID or agent ID must fail with a clear error.
- Invalid YAML or JSON schema must fail validation before running.
- Missing `.cogni/` in the current or parent directories must fail with a clear "run cogni init" style message.
- Missing `LLM_API_KEY` must fail fast.
- Commit ranges with missing runs must warn and continue.
- Citation validation failures must mark a task as failed.
- Budget overruns must mark a task as failed with `budget_exceeded`.
