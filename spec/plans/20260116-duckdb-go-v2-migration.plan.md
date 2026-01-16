# Plan: Migrate to duckdb-go/v2 + speed up bulk inserts

Date: 2026-01-16  
Status: TODO

## Goal
- Migrate from `github.com/marcboeker/go-duckdb` to the official `github.com/duckdb/duckdb-go/v2` driver.
- Find and fix inefficient bulk inserts (especially fixture loaders) to improve Tier C runtime.

## Non-goals
- Changing the DuckDB schema or test semantics.
- Rewriting WASM Tier D logic beyond any driver-related compatibility.
- Adding new user-facing features.

## Decisions
- Stay on `database/sql` for the general API surface, but use DuckDB Appenders for high-volume inserts.
- Keep fixture data deterministic and identical to current data generation.
- Preserve test timeouts and justfile tier commands.

## Key source snippets (current state)

`go.mod` (driver dependency)
```go
// go.mod
require (
	github.com/marcboeker/go-duckdb v1.8.5 // indirect
)
```

`internal/duckdb/testing/db.go` (driver import + sql.Open)
```go
// internal/duckdb/testing/db.go
import (
	"database/sql"
	_ "github.com/marcboeker/go-duckdb"
)

conn, err := sql.Open("duckdb", dsn)
```

`internal/duckdb/tier_c_fixtures_test.go` (bulk insert loop)
```go
// internal/duckdb/tier_c_fixtures_test.go
for _, runID := range runIDs {
	for metricIndex, metricID := range metricIDs {
		value := int64(i + metricIndex)
		if _, err := measStmt.ExecContext(ctx, runID, contextID, metricID, value); err != nil {
			return fixtureData{}, err
		}
	}
}
```

`scripts/duckdb/generate_fixture.go` (bulk insert loop)
```go
// scripts/duckdb/generate_fixture.go
for _, runID := range runIDs {
	for metricIndex, metricID := range metricIDs {
		value := int64(i + metricIndex)
		if _, err := measStmt.ExecContext(ctx, runID, contextID, metricID, value); err != nil {
			return err
		}
	}
}
```

## Step 1: Audit + baseline

Work:
- Read the DuckDB Go client docs and migration notes for v2.
- Inventory all DuckDB driver imports/usages.
- Capture a baseline timing for fixture loading (Tier C medium).
- Record all findings in the status file

Tests:
- `go test -tags duckdbtierc -run TestTierCMediumPerformance -count=1 ./internal/duckdb`

## Step 2: Migrate to `duckdb-go/v2`

Work:
- Update `go.mod` and imports to `github.com/duckdb/duckdb-go/v2`.
- Verify the `database/sql` driver name remains `duckdb`.
- Run `go mod tidy` after changes.

Tests:
- `go test ./internal/duckdb/...`
- `go test ./scripts/duckdb/...`

## Step 3: Add high-volume insert path (Appender)

Work:
- Add a helper to create and use DuckDB Appenders from a `duckdb-go/v2` connection.
- Replace the measurement insert loop in `loadFixture` with an appender path.
- Consider appender usage for revisions/contexts if it yields meaningful gains.
- Keep the data identical to the previous SQL insert path.

Tests:
- `go test ./internal/duckdb/...`
- `go test -tags duckdbtierc -run TestTierCMediumPerformance -count=1 ./internal/duckdb`

## Step 4: Scan and optimize other bulk inserts

Work:
- Scan for repeated `ExecContext` inserts inside loops (tests/scripts/ingestion helpers).
- Apply either:
  - Appenders for very large loops, or
  - Multi-row inserts / prepared batch statements where appropriate.
- Re-run baseline timing to confirm improvement.

Tests:
- `go test ./internal/duckdb/...`
- `just duckdb-tier-c`

## Step 5: Update docs + notes

Work:
- Update any internal docs/spec notes that mention the old driver.
- Document appender usage in helper docstrings where added.

Tests:
- None (documentation only).

## Done criteria
- Code uses `github.com/duckdb/duckdb-go/v2` and builds/tests cleanly.
- Fixture loading uses an optimized bulk path (appender or equivalent).
- Tier C medium runtime improves vs baseline without changing semantics.
