# Configuration

## Config sources

- `.cogni/config.yml` inside a `.cogni/` folder
- Environment variables for LLM provider credentials

## Config location rules

- `cogni init` proposes `.cogni/` at the git repo root; if no git repo is found, it uses the current folder.
- `cogni init` asks for confirmation before writing the folder and files.
- `cogni init` prompts for a results folder (default `.cogni/results`) and writes it to `repo.output_dir`.
- If a git repo is detected, `cogni init` offers to add the results folder to the repo root `.gitignore`.
- All commands locate `.cogni/` by walking up parent directories from the current working directory.
- `.cogni/schemas/` lives next to the config file and is loaded relative to the `.cogni/` folder.

## Environment variables

- `LLM_PROVIDER` (MVP: `openrouter`)
- `LLM_API_KEY`
- `LLM_MODEL`

## Adapters and Cucumber evaluation

Use adapters to connect `cucumber_eval` tasks to either a test runner (Godog)
or manual expectations when no test suite exists.

## Example config

```yaml
version: 1
repo:
  output_dir: ".cogni/results"
  setup_commands:
    - "go mod download"

agents:
  - id: default
    type: builtin
    provider: "openrouter"
    model: "gpt-4.1-mini"
    max_steps: 25
    temperature: 0.0

default_agent: "default"

tasks:
  - id: auth_flow_summary
    type: qa
    agent: "default"
    prompt: >
      Explain how authorization is enforced for API requests.
    eval:
      validate_citations: true
    budget:
      max_tokens: 12000
      max_seconds: 120
```

## Example Cucumber config

```yaml
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
    agent: "default"
    adapter: godog_default
    features:
      - "spec/features/cli.feature"
```
