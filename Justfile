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
    GOMODCACHE=/home/tzanko/.cache/go-mod GOCACHE=/home/tzanko/.cache/go-build go build -o cogni ./cmd/cogni

# Run Go tests with cache paths that are writable in the sandbox.
test:
    GOMODCACHE=/home/tzanko/.cache/go-mod GOCACHE=/home/tzanko/.cache/go-build go test ./...
