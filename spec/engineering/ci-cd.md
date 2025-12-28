# CI/CD

## Pipeline overview

- Run `go test ./...`.
- Build the `cogni` binary.
- Optionally run `cogni validate` and a demo `cogni run`.
- Upload `results.json` and `report.html` artifacts.

## Checks and gates

- Tests must pass before build artifacts are published.

## Release process

- Build binaries for target OS/arch.
- Publish via GitHub Releases.
