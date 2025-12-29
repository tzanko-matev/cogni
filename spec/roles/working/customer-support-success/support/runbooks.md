# Runbooks

## Routine operations

- Run `cogni run` on main and store outputs under `output_dir`.
- Generate reports with `cogni report --range <start>..<end>`.
- Share `report.html` with stakeholders.

## Emergency operations

- If the provider is unavailable, rerun later or switch model once supported.
- Rotate `LLM_API_KEY` if compromised.
