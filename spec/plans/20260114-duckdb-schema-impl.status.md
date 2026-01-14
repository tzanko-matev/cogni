# Status: Implement DuckDB Schema + Test Suite

ID: 20260114-duckdb-schema-impl.status  
Created: 2026-01-14  
Status: IN PROGRESS

Linked plan: `spec/plans/20260114-duckdb-schema-impl.plan.md`

## Current status
- Step 4 complete: Tier A constraint, JSON/MAP, and invariant tests added.

## What was done so far
- Added DuckDB tooling (DuckDB CLI, C toolchain, nodejs) to `flake.nix`.
- Added `github.com/marcboeker/go-duckdb` dependency to `go.mod`.
- Added `internal/duckdb/schema.sql` and schema loader helpers.
- Added test helper package for DuckDB connections.
- Added canonical JSON + fingerprint helpers and upsert helpers.
- Added ingestion helper tests for canonicalization stability and idempotency.
- Added Tier A schema existence, constraint, JSON/MAP, and invariant tests.

## Next steps
- Implement v_points view contract tests.

## Relevant source files (current or planned)
- `flake.nix`
- `go.mod`
- `go.sum`
- `internal/duckdb/schema.sql`
- `internal/duckdb/*.go`
- `internal/duckdb/*_test.go`
- `tests/fixtures/duckdb/`
- `Justfile`
