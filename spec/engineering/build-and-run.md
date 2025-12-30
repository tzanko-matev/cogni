# Build and Run

## Build steps

- `go build ./cmd/cogni`

## Run locally

- `./cogni init`
- `./cogni validate`
- `./cogni run`
- `./cogni run --verbose`
- `./cogni run --verbose --no-color`
- `./cogni run task-id@agent-id`
- `./cogni compare --base main`
- `./cogni report --range main..HEAD --open`

## Common commands

- `cogni init` - scaffold config
- `cogni validate` - validate config
- `cogni run` - execute benchmark
- `cogni compare` - compare runs
- `cogni report` - generate report
