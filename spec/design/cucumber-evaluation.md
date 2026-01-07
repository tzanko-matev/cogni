# Cucumber Evaluation

## Goal
Enable Cogni to evaluate whether Cucumber feature examples are implemented by
comparing an agent's judgement against ground-truth results. Ground truth may
come from a test runner (Godog) or from manually curated expectations when a
test suite is unavailable.

## Feature file inputs
- Feature files are parsed with Gherkin to enumerate scenarios and example rows.
- Each example is assigned a stable Example ID. Avoid line numbers; prefer
  explicit identifiers in the feature file.

### Example ID rules (in priority order)
1. Scenario tag `@id:<scenario_id>` plus Examples column `id`:
   `<scenario_id>:<row_id>`
2. Scenario tag `@id:<scenario_id>` plus row index:
   `<scenario_id>:<row_index>`
3. Scenario name plus Examples column `id`:
   `<scenario_name>#<row_id>`
4. Fallback only: file path + line number + row index.

## Task type
Introduce a task type that evaluates feature examples.

- `type: cucumber_eval`
- `features`: list of feature files or globs to evaluate
- `adapter`: reference to a configured adapter
- `prompt_template`: prompt used per feature file (batch examples)

### Prompt template placeholders
The prompt template must include the feature context and the expected example
IDs for the feature file being evaluated.

- `{feature_path}`: normalized path to the feature file.
- `{feature_text}`: full contents of the feature file.
- `{example_ids}`: newline-delimited list of expected Example IDs for the file.

### Agent response schema (batch)
The agent returns a single JSON object per feature file:

```json
{
  "results": [
    {
      "example_id": "scenario_id:row_id",
      "implemented": true,
      "evidence": [{"path": "internal/foo.go", "lines": [10, 11]}],
      "notes": "optional"
    }
  ]
}
```

Validation rules:
- Every result must include a non-empty `example_id`.
- Example IDs must be unique.
- The set of IDs must exactly match the expected Example IDs for the feature
  file. Missing or extra IDs are errors.

## Adapter types

### Godog adapter (test-based)
Runs Godog and maps test results to Example IDs.

- `type: cucumber`
- `runner: godog`
- `formatter: json`
- `feature_roots`: default search roots
- `tags`: optional tag filter

Status mapping:
- `passed` -> implemented
- `failed`, `undefined`, `pending`, `skipped` -> not implemented

### Manual expectations adapter
Uses a curated expectations file instead of running tests.

- `type: cucumber_manual`
- `feature_roots`: default search roots
- `expectations_dir`: location of expectation files
- `match`: options like `require_evidence`, `normalize_whitespace`

Expectation files map Example IDs to expected outcomes.

## Config example

```yaml
version: 1
repo:
  output_dir: "./cogni-results"

agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"

default_agent: "default"

adapters:
  - id: godog_default
    type: cucumber
    runner: godog
    formatter: json
    feature_roots:
      - "spec/features"

  - id: manual_expectations
    type: cucumber_manual
    feature_roots:
      - "spec/features"
    expectations_dir: "spec/expectations"

tasks:
  - id: cucumber_cli_features
    type: cucumber_eval
    agent: default
    adapter: godog_default
    features:
      - "spec/features/cli.feature"
    prompt_template: |
      You are evaluating the following Cucumber feature file:
      Path: {feature_path}

      Feature text:
      {feature_text}

      Expected Example IDs (one per line):
      {example_ids}

      For each example ID, decide if the behavior is fully implemented.
      Return ONLY JSON:
      {"results":[{"example_id":"...","implemented":true,"evidence":[{"path":"...","lines":[1,2]}],"notes":"..."}]}
```

## Outputs
For each example, `results.json` records:
- Example ID
- Agent decision and evidence
- Ground-truth status (test-based or manual expectation)
- Correct/incorrect flag

For each feature file, `results.json` also records feature-level effort metrics
(tokens, wall time, tool calls).

The CLI summary reports counts and accuracy per task, plus overall totals.
