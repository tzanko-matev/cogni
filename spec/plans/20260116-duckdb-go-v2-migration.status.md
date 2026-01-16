# Status: Migrate to duckdb-go/v2 + speed up bulk inserts

ID: 20260116-duckdb-go-v2-migration.status  
Created: 2026-01-16  
Status: IN PROGRESS

Linked plan: `spec/plans/20260116-duckdb-go-v2-migration.plan.md`

## Current status
- IN PROGRESS. Driver migration complete; bulk insert optimization and doc updates remain.

## What was done so far
- Captured baseline: `TestTierCMediumPerformance` timed out at the 30s deadline (2026-01-16).
- Migrated driver imports and modules to `github.com/duckdb/duckdb-go/v2` (v2.5.4).
- Aligned MAP extraction test SQL with `MAP(VARCHAR, VARCHAR)` semantics.
- Tests: `go test ./internal/duckdb/...` (pass).

## Next steps
- Add Appender-based inserts for measurement fixtures and update helper scripts.
- Re-run Tier C medium timing to confirm improvement.
- Update docs/plan status to DONE.

## Relevant source files (current or planned)
- `go.mod`
- `go.sum`
- `internal/duckdb/testing/db.go`
- `internal/duckdb/tier_c_fixtures_test.go`
- `scripts/duckdb/generate_fixture.go`
- `internal/duckdb/` (other insert-heavy paths)
