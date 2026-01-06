# Acceptance Criteria

## Definition of done

- `.cogni/config.yml` supports agents, tasks, and `output_dir`.
- `.cogni/schemas/` is created by `cogni init` and loaded for validation.
- `cogni init` confirms the target `.cogni/` location before writing.
- `cogni init` prompts for a results folder (default `.cogni/results`) and writes it to `repo.output_dir`.
- In a git repo, `cogni init` offers to add the results folder to the repo root `.gitignore`.
- CLI commands find `.cogni/` by walking up parent directories.
- `cogni validate` rejects invalid YAML or schemas.
- `cogni run` produces `results.json` and `report.html` under the configured output directory.
- `cogni run task-id@agent-id` uses the specified agent.
- `cogni run` supports `cucumber_eval` tasks with Godog or manual expectations.
- `cogni compare --base main` resolves refs and prints deltas.
- `cogni report --range main..HEAD` renders trend charts from the commit window.

## Testable outcomes

- QA tasks fail on invalid JSON, schema mismatch, or invalid citations.
- Results include VCS type, agent ID, model, and metrics per attempt.
- Range queries warn when runs are missing but still render remaining data.
- `results.json` includes per-example verdicts and accuracy for `cucumber_eval` tasks.
