# API Design

## Surface area

- CLI commands: `cogni init`, `cogni validate`, `cogni run`, `cogni compare`, `cogni report`
- Configuration: `.cogni.yml` and JSON schemas

## Endpoints or interfaces

- `cogni run [task-id|task-id@agent-id]...`
- `cogni compare --base <commit|run-id|ref> [--head <commit|run-id|ref>]`
- `cogni compare --range <start>..<end>`
- `cogni report --range <start>..<end>`

## Run flags

- `cogni run --verbose`: stream detailed execution logs to the console (LLM input/output, tool calls and results, per-task metrics) with ANSI styling when stdout is a terminal.
- `cogni run --no-color`: disable ANSI styling for verbose console logs.

## Request and response examples

```bash
cogni run
cogni run --verbose
cogni run --verbose --no-color
cogni run auth_flow_summary@default
cogni compare --base main
cogni report --range main..HEAD --open
```

Outputs:

- `results.json` per run
- `report.html` per run or range
- CLI summary output for compare/report

## Error handling

- Non-zero exit codes on invalid config, missing API keys, or invalid task selectors.
- Clear messages for missing runs or invalid ranges.
