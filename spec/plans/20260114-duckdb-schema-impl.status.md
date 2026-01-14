# Status: Implement DuckDB Schema + Test Suite

ID: 20260114-duckdb-schema-impl.status  
Created: 2026-01-14  
Status: IN PROGRESS

Linked plan: `spec/plans/20260114-duckdb-schema-impl.plan.md`

## Current status
- Step 2 complete: schema DDL + loader helpers added.

## What was done so far
- Added DuckDB tooling (DuckDB CLI, C toolchain, nodejs) to `flake.nix`.
- Added `github.com/marcboeker/go-duckdb` dependency to `go.mod`.
- Added `internal/duckdb/schema.sql` and schema loader helpers.
- Added test helper package for DuckDB connections.

## Next steps
- Implement ingestion helpers + unit tests.

## Relevant source files (current or planned)
- `flake.nix`
- `go.mod`
- `go.sum`
- `internal/duckdb/schema.sql`
- `internal/duckdb/*.go`
- `internal/duckdb/*_test.go`
- `tests/fixtures/duckdb/`
- `Justfile`
