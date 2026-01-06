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
- `prompt_template`: prompt used per example

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
      Read the source code. For the example {example_id}, decide if it is fully
      implemented. List relevant file names and line numbers.
      Return JSON with example_id, implemented, evidence, notes.
```

## Outputs
For each example, `results.json` records:
- Example ID
- Agent decision and evidence
- Ground-truth status (test-based or manual expectation)
- Correct/incorrect flag

The CLI summary reports counts and accuracy per task, plus overall totals.
