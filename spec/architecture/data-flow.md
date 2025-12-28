# Data Flow

## Data lifecycle

- Read `.cogni.yml` and schemas.
- Resolve commit(s) to evaluate.
- Execute QA tasks through the agent.
- Produce per-attempt metrics and evaluation artifacts.
- Write `results.json` and `report.html` under `output_dir`.

## Data stores

- Local filesystem under `output_dir`.

## Data retention

- User-managed; `cogni` does not delete data.
- Reports are regenerated on demand from stored results.
