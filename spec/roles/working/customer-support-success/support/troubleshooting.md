# Troubleshooting

## Common issues

- Missing `LLM_API_KEY` or invalid provider configuration.
- Invalid `.cogni/config.yml`, missing `.cogni/schemas/`, or no `.cogni/` found in parent directories.
- Unknown task ID or agent ID.
- No results found for a commit range.
- Missing Question Spec files or invalid `<answer>` output formats.

## Diagnostics

- Run `cogni validate` to check config.
- Inspect `results.json` and per-task logs.
- Confirm `rg` is installed and on PATH.
