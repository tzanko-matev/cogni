# Build and Run

## Build steps

- `go build ./cmd/cogni`

## Run locally

- `./cogni init`
- `./cogni validate`
- `./cogni run`
- `./cogni run task-id@agent-id`
- `./cogni compare --base main`
- `./cogni report --range main..HEAD --open`
- Commands can run from subdirectories; `.cogni/` is discovered by walking up parent directories.

## Common commands

- `cogni init` - scaffold `.cogni/` (prompts for location, results folder, and `.gitignore` in git repos)
- `cogni validate` - validate config
- `cogni run` - execute benchmark
- `cogni compare` - compare runs
- `cogni report` - generate report
