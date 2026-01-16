# Status: Migrate to duckdb-go/v2 + speed up bulk inserts

ID: 20260116-duckdb-go-v2-migration.status  
Created: 2026-01-16  
Status: TODO

Linked plan: `spec/plans/20260116-duckdb-go-v2-migration.plan.md`

## Current status
- Not started. Plan created.

## What was done so far
- None.

## Next steps
- Start Step 1 (audit + baseline) from the plan.

## Relevant source files (current or planned)
- `go.mod`
- `go.sum`
- `internal/duckdb/testing/db.go`
- `internal/duckdb/tier_c_fixtures_test.go`
- `scripts/duckdb/generate_fixture.go`
- `internal/duckdb/` (other insert-heavy paths)
