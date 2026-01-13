# Plan: Question Spec Core Evaluation

Date: 2026-01-09
Status: DONE

## Goal
Replace the current cucumber-focused evaluation with a single, minimal core: evaluate a Question Spec (JSON/YAML) by asking an agent each question and comparing the final XML `<answer>` against known correct answers. Provide a direct CLI workflow (`cogni eval <questions_file> --agent <id>`) while keeping config-defined tasks runnable via `cogni run`.

## Non-Goals
- No Cucumber adapters or cucumber_eval tasks in this iteration (they are removed).
- No multi-answer XML format (`<answers>`); only `<answer>`.
- No semantic similarity or fuzzy matching beyond trim + case-insensitive comparison.
- No batching multiple questions into a single agent call.

## UX Summary
- **CLI (direct):** `cogni eval questions.yml --agent default_agent`
  - Loads `.cogni` config to resolve agent definitions.
  - Produces the same results/report outputs as `cogni run`.
- **CLI (config tasks):** `cogni run [task-id...]`
  - Supports `type: question_eval` tasks that reference `questions_file`.

## Question Spec Format
Accept JSON or YAML:
```yaml
version: 1
questions:
  - id: q1
    question: What is 2+2?
    answers: ["3", "4", "5"]
    correct_answers: ["4"]
```

Validation rules:
- `questions` non-empty.
- `answers` non-empty strings.
- `correct_answers` non-empty and subset of `answers`.
- Optional `id` must be unique if provided.

## Agent Output Contract
The agent may provide rationale, but the **final output must end with**:
```
<answer>...</answer>
```
Rules:
- Only `<answer>` is allowed (no `<answers>`).
- No trailing text after `</answer>`.
- Parsing uses `encoding/xml` on trailing XML only.
- Matching is trim + case-insensitive; if multiple correct answers exist, any single match passes.

## Deprecated Logic Removal
Remove cucumber-related logic and docs:
- `internal/cucumber/*`
- `internal/runner/cucumber*`
- cucumber-specific result fields and summaries
- cucumber feature tests + docs (spec/features, requirements, design docs)
- adapter config and validation

## Implementation Steps
1. **Config Schema Cleanup**
   - Remove `Adapters`/`AdapterConfig` from `spec.Config`.
   - Add `questions_file` to `spec.TaskConfig`.
   - Update config validation to accept only `question_eval` task types.
2. **Question Spec Loader**
   - Add a new package (e.g. `internal/question`) to load JSON/YAML.
   - Validate `answers`/`correct_answers`, normalize for matching.
3. **Question Eval Runner**
   - Implement `runQuestionTask` in `internal/runner`.
   - For each question: build prompt, run agent, parse trailing `<answer>`, score.
4. **XML Extraction + Parsing**
   - Implement a helper to extract trailing XML `<answer>` and parse.
   - Record parse errors and mark incorrect when invalid.
5. **Results Model**
   - Introduce `QuestionEval`, `QuestionResult`, and summary fields in `internal/runner/results.go`.
   - Remove cucumber result structs and summary aggregation.
6. **CLI**
   - Add `cogni eval` command with `--agent`, `--spec`, `--output-dir`, `--verbose`, `--log`, `--no-color`.
   - Ensure it loads config for agent definitions and uses question spec file as input.
7. **Report + Summary**
   - Update HTML report to show question results + accuracy.
   - Update CLI run summary output for question_eval tasks.
8. **Docs**
   - Add `spec/design/question-evaluation.md`.
   - Update `spec/engineering/configuration.md` and role guides to reflect new core.
   - Remove or archive cucumber docs.
9. **Tests**
   - Unit tests for spec loader (JSON + YAML).
   - XML extraction/parsing tests.
   - Runner tests for evaluation and summary.
   - CLI tests for `cogni eval`.

## Testing Policy
- Each implementation step is considered complete only when the relevant tests exist and pass.

## Files to Touch (Expected)
- `internal/runner/*` (new question eval flow, results, summaries)
- `internal/cli/*` (new eval command)
- `internal/config/*` (schema + validation updates)
- `internal/report/*` (question results in HTML)
- `internal/spec/types.go`
- `spec/design/question-evaluation.md`
- `spec/engineering/configuration.md`
- Remove `internal/cucumber/*`, `tests/cucumber/*`, cucumber feature specs

## Risks
- Breaking changes from removing cucumber_eval/adapters; update docs + CLI messaging.
- XML parsing edge cases; enforce “final XML only” rule to avoid ambiguity.

## Done Criteria
- `cogni eval` runs a question spec and writes results/report.
- `cogni run` supports `question_eval` tasks.
- Cucumber logic removed from code and docs.
- Tests updated and passing.
