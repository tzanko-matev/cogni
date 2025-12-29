# Development Setup

## Prerequisites

- Go 1.22+
- git
- Jujutsu (`jj`) for local version control
- ripgrep (`rg`)
- OpenRouter API key (`LLM_API_KEY`)

## Local setup steps

- `go mod download`
- Set env vars:
  - `LLM_PROVIDER=openrouter`
  - `LLM_MODEL=<model>`
  - `LLM_API_KEY=<key>`

## Version control workflow

- Use `jj status`, `jj log`, and `jj diff` to inspect changes.
- Start work with `jj new` and label it with `jj describe -m "message"`.
- Push to git remotes with `jj git push` when needed.
