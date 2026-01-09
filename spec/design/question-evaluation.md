# Question Evaluation

## Overview

Question evaluation (`question_eval`) measures how well an agent can answer a curated set of
repository questions. Each question is answered in a separate agent run, and the final
`<answer>` XML block is compared to the list of correct answers.

## Question Spec format

Question specs are JSON or YAML files with the following schema:

```yaml
version: 1
questions:
  - id: q1
    question: What is 2+2?
    answers: ["3", "4", "5"]
    correct_answers: ["4"]
```

Rules:

- `questions` must be non-empty.
- `answers` and `correct_answers` must contain non-empty strings.
- `correct_answers` must be a subset of `answers` (case-insensitive, trimmed).
- `id` is optional but must be unique if present.

## Agent output contract

Agents may include reasoning, but the response must end with:

```
<answer>...</answer>
```

Only a single `<answer>` block is supported. No trailing text may appear after the closing tag.

## Evaluation flow

1. Load and validate the Question Spec.
2. For each question:
   - Build the question prompt with answer choices.
   - Run the agent once.
   - Extract and parse the trailing `<answer>` XML.
   - Compare the normalized answer against `correct_answers`.
3. Aggregate per-question accuracy into the task and run summary.

## Results

`results.json` captures:

- Per-question verdicts, parsed answers, and errors.
- Effort metrics per question (tokens, wall time, steps, tool calls).
- Task-level accuracy summary.
