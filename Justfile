set shell := ["bash", "-cu"]
cache_dir := justfile_directory() + "/.cache"
go_mod_cache := cache_dir + "/go-mod"
go_build_cache := cache_dir + "/go-build"

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
    GOMODCACHE={{go_mod_cache}} GOCACHE={{go_build_cache}} go build -o cogni ./cmd/cogni

# Run Go tests with cache paths that are writable in the sandbox.
test:
    GOMODCACHE={{go_mod_cache}} GOCACHE={{go_build_cache}} go test ./...

# Run live-key integration tests.
test-live:
    GOMODCACHE={{go_mod_cache}} GOCACHE={{go_build_cache}} go test -tags=live -timeout 10m ./internal/cli

# Run cucumber feature tests.
test-cucumber:
    GOMODCACHE={{go_mod_cache}} GOCACHE={{go_build_cache}} go test -tags=cucumber -timeout 2m ./tests/cucumber
