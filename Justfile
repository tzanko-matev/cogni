set shell := ["bash", "-cu"]

# Serve the Hugo docs site from spec/roles.
docs-serve:
    hugo server --bind 0.0.0.0 --port 1313

# Generate Hugo data from the latest Go test results.
docs-test-results:
    ./scripts/generate-test-results.sh

# Serve docs with fresh test results.
docs-serve-with-tests: docs-test-results
    hugo server --bind 0.0.0.0 --port 1313

# Build the cogni CLI.
build:
    go generate ./...
    go build -o cogni ./cmd/cogni

# Run Go tests with cache paths that are writable in the sandbox.
test:
    go test ./...

# Run live-key integration tests.
test-live:
    go test -tags=live -timeout 10m ./internal/cli
