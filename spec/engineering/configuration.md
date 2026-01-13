# Configuration

## Config sources

- `.cogni.yml` in the repo
- Environment variables for LLM provider credentials

## Environment variables

- `LLM_PROVIDER` (MVP: `openrouter`)
- `LLM_API_KEY`
- `LLM_MODEL`

## Question evaluation

Question evaluation tasks (`question_eval`) load a Question Spec (JSON or YAML).
Each task references `questions_file` and runs each question through the selected agent.

## Compaction settings

Tasks may include a `compaction` block to configure soft limits and summarization:

- `compaction.max_tokens`: soft limit to trigger auto-compaction (defaults to a fraction of the task budget).
- `compaction.summary_prompt` or `compaction.summary_prompt_file`: optional summary prompt override.
- `compaction.recent_user_token_budget`: token budget for keeping recent user messages.
- `compaction.recent_tool_output_limit`: number of tool outputs to retain during compaction.

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
  - id: question_eval_core
    type: question_eval
    agent: "default"
    questions_file: "spec/questions/core.yml"
    budget:
      max_tokens: 12000
      max_seconds: 120
    compaction:
      max_tokens: 9000
      recent_user_token_budget: 2000
      recent_tool_output_limit: 3
      summary_prompt_file: "prompts/compaction_summary.txt"
```

## Example question evaluation config

```yaml
tasks:
  - id: question_eval_core
    type: question_eval
    agent: "default"
    questions_file: "spec/questions/core.yml"
```
