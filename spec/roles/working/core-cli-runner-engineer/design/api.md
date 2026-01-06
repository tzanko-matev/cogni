# API Design

## Surface area

- CLI commands: `cogni init`, `cogni validate`, `cogni run`, `cogni compare`, `cogni report`
- Configuration: `.cogni/config.yml` and `.cogni/schemas/` under `.cogni/`
- Task types: `qa`, `cucumber_eval`
- Adapters: `cucumber` (Godog) and `cucumber_manual` (manual expectations)

## Init behavior

- `cogni init` proposes a `.cogni/` location (git repo root if available, otherwise the current directory).
- The user must confirm the proposed location before files are written.
- `cogni init` asks for a results folder (default `.cogni/results`) and writes it to `repo.output_dir`.
- If a git repo is detected, `cogni init` offers to add the results folder to the repo root `.gitignore`.
- All commands resolve `.cogni/` by walking up parent directories from the current working directory.

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
- Per-example verdicts for `cucumber_eval` tasks

## Error handling

- Non-zero exit codes on invalid config, missing API keys, or invalid task selectors.
- Clear messages for missing runs or invalid ranges.
- Actionable errors for missing feature files, expectations, or Godog failures.
