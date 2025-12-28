# Configuration

## Config sources

- `.cogni.yml` in the repo
- Environment variables for LLM provider credentials

## Environment variables

- `LLM_PROVIDER` (MVP: `openrouter`)
- `LLM_API_KEY`
- `LLM_MODEL`

## Example config

```yaml
version: 1
repo:
  output_dir: "./cogni-results"
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
