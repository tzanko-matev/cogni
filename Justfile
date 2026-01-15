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

# Build report UI assets.
web-build:
    mkdir -p .cache/npm
    if [ ! -d web/node_modules ]; then npm --cache .cache/npm --prefix web install; fi
    npm --cache .cache/npm --prefix web run build

# Sync built report assets into the embedded reportserver directory.
web-sync-assets:
    mkdir -p .cache/npm
    if [ ! -d web/node_modules ]; then npm --cache .cache/npm --prefix web install; fi
    npm --cache .cache/npm --prefix web run build
    rm -rf internal/reportserver/assets
    mkdir -p internal/reportserver/assets
    cp -R web/dist/. internal/reportserver/assets/

# Build the cogni CLI.
build: web-build
    go generate ./...
    go build -o cogni ./cmd/cogni

# Run Go tests with cache paths that are writable in the sandbox.
test:
    go test ./...

# Run live-key integration tests.
test-live:
    go test -tags=live -timeout 10m ./internal/cli

# Run cucumber feature tests.
test-cucumber:
    go test -tags=cucumber ./...

# Run integration-tagged tests.
test-integration:
    go test -tags=integration ./...

# Run stress-tagged tests.
test-stress:
    go test -tags=stress ./...

# Run chaos-tagged tests.
test-chaos:
    go test -tags=chaos ./...

# Run stress tests that require both stress + integration tags.
test-stress-integration:
    go test -tags=stress,integration ./internal/stress

# Run chaos tests that require both chaos + integration tags.
test-chaos-integration:
    go test -tags=chaos,integration ./internal/chaos

# Run python unit tests.
test-python:
    python -m pytest tests/entropy_agent

# Run all test suites (unit + tagged).
test-all: test test-live test-cucumber test-integration test-stress test-chaos test-stress-integration test-chaos-integration test-python duckdb-tier-all

# Run DuckDB Tier B fuzz/property tests.
duckdb-tier-b:
    go test ./internal/duckdb -run 'TestCanonicalJSONFuzzStability|TestFingerprintCollisionFuzz'

# Run DuckDB Tier C medium fixture performance + durability tests.
duckdb-tier-c:
    go test -tags=duckdbtierc ./internal/duckdb -run 'TestTierC'

# Run DuckDB Tier C large fixture stress test (optional).
duckdb-tier-c-large:
    go test -tags=duckdbtierc,duckdbtierclarge ./internal/duckdb -run 'TestTierCLarge'

# Run DuckDB Tier D DuckDB-WASM smoke test.
duckdb-tier-d:
    go run ./scripts/duckdb/... --config tests/fixtures/duckdb/medium.json --out tests/fixtures/duckdb/medium.duckdb
    if [ ! -d tests/duckdb/wasm/node_modules ]; then (cd tests/duckdb/wasm && npm install); fi
    node tests/duckdb/wasm/smoke_test.mjs tests/fixtures/duckdb/medium.duckdb

# Run all DuckDB test tiers.
duckdb-tier-all: duckdb-tier-b duckdb-tier-c duckdb-tier-c-large duckdb-tier-d
