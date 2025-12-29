# Data Model

## Entities

- Repo: name, VCS type, commit, branch
- Agent: id, type, provider, model, settings
- Task: id, type, prompt, evaluation rules, budget, agent selection
- Run: run_id, timestamps, tasks, summary
- Attempt: metrics and evaluation results for a task execution

## Relationships

- A run contains many tasks.
- Each task references an agent ID.
- Each task can have multiple attempts when `--repeat` is used.

## Schemas

- `.cogni.yml` defines repo settings, agents, tasks, and budgets.
- `results.json` captures run metadata, agent definitions, per-task attempts, and summaries.
