# Data Model

## Entities

- Repo: name, VCS type, commit, branch
- Agent: id, type, provider, model, settings
- Task: id, type, prompt, evaluation rules, budget, agent selection
- Question: a single question with answer choices and correct answers
- QuestionSpec: a JSON/YAML document containing questions
- Run: run_id, timestamps, tasks, summary
- Attempt: metrics and evaluation results for a task execution

## Relationships

- A run contains many tasks.
- Each task references an agent ID.
- Each task can have multiple attempts when `--repeat` is used.
- `question_eval` tasks load Questions from a QuestionSpec file.
- Question verdicts are computed by comparing the parsed `<answer>` to correct answers.

## Schemas

- `.cogni.yml` defines repo settings, agents, tasks, and budgets.
- `results.json` captures run metadata, agent definitions, per-task attempts, and summaries.
- `results.json` includes per-question verdicts and accuracy for `question_eval` tasks.
