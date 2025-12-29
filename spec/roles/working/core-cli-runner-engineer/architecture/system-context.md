# System Context

## Context diagram

- User or CI invokes the `cogni` CLI in a git repo.
- `cogni` reads repo files, sends questions to the LLM provider, and writes results locally.

## External systems

- Git repository (local workspace)
- OpenRouter API (LLM provider in MVP)
- Local filesystem for outputs
- CI runner (optional)

## Interfaces

- CLI commands (`cogni init|validate|run|compare|report`)
- `.cogni/config.yml` configuration and `.cogni/schemas/` (discovered by walking up parent directories)
- Environment variables (`LLM_PROVIDER`, `LLM_API_KEY`, `LLM_MODEL`)
