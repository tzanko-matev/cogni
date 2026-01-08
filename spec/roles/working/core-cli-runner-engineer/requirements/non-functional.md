# Non-Functional Requirements

## Performance

- Use `rg` for search to keep query latency low.
- Enforce per-task budgets for tokens, steps, and wall time.
- Limit file read sizes and tool output volume.

## Reliability

- Execute tasks in a deterministic order.
- Write `results.json` even when some tasks fail.
- Keep outputs stable for identical inputs and settings.

## Security

- Read-only tooling only (list_files/list_dir/search/read_file).
- No code modifications to the repo in MVP.
- Results stored locally; no uploads to hosted services.
- Treat `.cogni/config.yml` and `.cogni/schemas/` as trusted within the repo boundary.

## Compliance

- No explicit compliance requirements in MVP.

## Accessibility

- Reports must be readable in a standard browser.
- Charts and tables should be legible without zooming.
