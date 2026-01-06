# Data Model

## Entities

- Repo: name, VCS type, commit, branch
- Agent: id, type, provider, model, settings
- Task: id, type, prompt, evaluation rules, budget, agent selection
- Adapter: id, type, runner, feature roots, expectations location
- Example: a Cucumber scenario or scenario outline row
- Expectation: curated expected status and evidence for an example
- Run: run_id, timestamps, tasks, summary
- Attempt: metrics and evaluation results for a task execution

## Relationships

- A run contains many tasks.
- Each task references an agent ID.
- Each task can have multiple attempts when `--repeat` is used.
- `cucumber_eval` tasks enumerate Examples from feature files via an Adapter.
- Example verdicts are computed by comparing agent decisions to ground truth.

## Schemas

- `.cogni.yml` defines repo settings, agents, tasks, and budgets.
- `results.json` captures run metadata, agent definitions, per-task attempts, and summaries.
- `results.json` includes per-example verdicts for `cucumber_eval` tasks.
