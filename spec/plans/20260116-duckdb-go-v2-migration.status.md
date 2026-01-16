# Status: Migrate to duckdb-go/v2 + speed up bulk inserts

ID: 20260116-duckdb-go-v2-migration.status  
Created: 2026-01-16  
Status: DONE

Linked plan: `spec/plans/20260116-duckdb-go-v2-migration.plan.md`

## Current status
- DONE. Migrated to duckdb-go/v2 and optimized bulk inserts with Appenders.

## What was done so far
- Captured baseline: `TestTierCMediumPerformance` timed out at the 30s deadline (2026-01-16).
- Migrated driver imports and modules to `github.com/duckdb/duckdb-go/v2` (v2.5.4).
- Aligned MAP extraction test SQL with `MAP(VARCHAR, VARCHAR)` semantics.
- Added DuckDB Appender-based inserts for measurement fixtures in tests and scripts.
- Added UUID conversion helpers for appender rows.
- Updated docs that referenced the old driver.
- Tests: `go test ./internal/duckdb/...` (pass), `go test -tags duckdbtierc -run TestTierCMediumPerformance -count=1 ./internal/duckdb` (pass; total ~3.5s on 2026-01-16), `just duckdb-tier-d` (pass).

## Next steps
- None.

## Relevant source files (current or planned)
- `go.mod`
- `go.sum`
- `internal/duckdb/testing/db.go`
- `internal/duckdb/tier_c_fixtures_test.go`
- `scripts/duckdb/generate_fixture.go`
- `internal/duckdb/` (other insert-heavy paths)
