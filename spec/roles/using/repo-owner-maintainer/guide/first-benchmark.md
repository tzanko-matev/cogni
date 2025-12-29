# First benchmark

## Goal
Set up a repeatable benchmark for your repo with clear outputs you can share.

## Prereqs
- Go 1.22+
- git
- ripgrep (`rg`)
- `LLM_PROVIDER`, `LLM_MODEL`, and `LLM_API_KEY` configured

## Define questions and agents
- Run `cogni init` and confirm the suggested `.cogni/` location (repo root if in git, otherwise the current folder).
- Choose a results folder when prompted (default `.cogni/results`); `cogni init` writes it to `repo.output_dir` in `.cogni/config.yml`.
- If a git repo is detected, `cogni init` offers to add the results folder to the repo root `.gitignore`.
- `cogni init` creates `.cogni/config.yml` and `.cogni/schemas/`.
- Add `qa` tasks with prompts tied to product features.
- Require citations so answers are traceable to code.
- Set `output_dir` once so run commands stay short.

## Run and review
- `cogni validate` to fail fast on config errors.
- `cogni run` for a full run at the current commit.
- `cogni compare --base main` to see deltas against main.
- `cogni report --range main..HEAD` for trend charts.
- You can run commands from any subdirectory; Cogni finds `.cogni/` by walking up parent directories.

## Outputs
- `results.json` and `report.html` are written under `<output_dir>/<commit>/<run-id>/`.
- Reports show pass rate, tokens, and wall time trends.

## See also
- [spec/roles/working/core-cli-runner-engineer/engineering/configuration.md](/working/core-cli-runner-engineer/engineering/configuration/)
- [spec/roles/working/core-cli-runner-engineer/engineering/build-and-run.md](/working/core-cli-runner-engineer/engineering/build-and-run/)
- [spec/roles/working/customer-support-success/support/troubleshooting.md](/working/customer-support-success/support/troubleshooting/)
