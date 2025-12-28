# Security Architecture

## Threat model

- Untrusted prompts could attempt data exfiltration via tool calls.
- LLM provider access requires API keys.
- Local results may include sensitive metadata.

## Authn and authz

- None in MVP (local CLI only).

## Data protection

- Read-only tooling; no code modifications to the repo.
- Results stored locally; no hosted uploads in MVP.
- Keep tool output sizes bounded to reduce accidental leakage.

## Secrets management

- Use environment variables for `LLM_API_KEY`.
- Do not persist secrets in results or reports.
