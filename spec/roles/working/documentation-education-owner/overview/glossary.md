# Glossary

- Cognitive benchmark: A repeatable suite of questions used to measure codebase understanding.
- Cognitive test: A single question answered by an agent with citations.
- Probe question: A repo-specific question tied to a product feature.
- Config dir: The `.cogni/` folder that contains `config.yml` and `schemas/`.
- Task: A `qa` or `question_eval` item in `.cogni/config.yml`.
- Run: An execution of tasks at a specific commit.
- Attempt: A single execution of a task (supports repeats).
- Agent: The configured system that answers questions.
- Agent ID: Identifier for an agent configuration in `.cogni/config.yml`.
- Model: The LLM model used by an agent.
- Provider: The LLM provider (MVP: OpenRouter).
- Results JSON: `results.json` output for a run.
- Report HTML: `report.html` generated from results.
- Output dir: Configured folder where results and reports are stored (default `.cogni/results`).
- Range: A commit window (e.g., `main..HEAD`) used for trends.
- Pass rate: Ratio of passing tasks to total tasks in a run.
- Question spec: A JSON or YAML file defining questions, answer choices, and correct answers.
