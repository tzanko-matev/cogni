# Implementation Plan (v1)

Each step must include tests with explicit timeouts. Use `.feature` files where
behavior is user-visible (for schema invariants, we still provide a minimal
feature file for documentation and for godog if desired). All tests are run
manually via `just` commands.

## Step 0: Dev environment update

Work:
- Add `duckdb` to the Nix dev shell for the CLI and library.
- Add a C toolchain for CGO if we use `go-duckdb`:
  - `pkg-config`
  - `clang` (or `gcc`)
- Add `nodejs` for DuckDB-WASM smoke tests.
- Export `DUCKDB_BIN` in the shell hook (optional; helps sqllogictest).

Tests:
- None (environment-only change).

## Step 1: Add schema DDL + loader

Work:
- Create `internal/duckdb/schema.sql` with the DDL from `schema.md`.
- Add a Go helper `EnsureSchema(db *sql.DB) error` that executes the SQL.
- Keep DDL in one file to reduce drift between tests and production.

Tests:
- `nix develop -c go test ./internal/duckdb/...` (timeout <= 2s per test)

## Step 2: Add ingestion helpers (keys + upserts)

Work:
- Implement `CanonicalJSON` and `FingerprintJSON` helpers.
- Add `UpsertAgent`, `UpsertQuestion`, `UpsertContext` helpers that return IDs.
- Ensure deterministic `context_key` construction.

Tests:
- Go unit tests for canonicalization stability (timeout <= 1s).
- Go unit tests for upsert idempotency (timeout <= 2s).

## Step 3: Constraint + invariant tests (Tier A)

Work:
- Add tests for primary key and unique constraints.
- Add invariant queries for value-column correctness and orphan checks.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test)

## Step 4: v_points view contract tests

Work:
- Add fixtures that cover numeric and non-numeric metrics.
- Assert `v_points` shape and value semantics.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test)

## Step 5: Tier B property-based tests

Work:
- Add `testing/quick`-based generators for agent/question/context specs.
- Persist failing seeds to `tests/fixtures/duckdb/fuzz/`.

Tests:
- `just duckdb-tier-b` (timeout <= 5s per test file)

## Step 6: Tier C performance + durability tests

Work:
- Add a benchmark runner that loads the medium fixture and runs the core
  report queries, recording wall time.
- Add optional large fixture run for stress testing.
- Add crash-safety tests on an on-disk DB (create, insert, kill before commit).

Tests:
- `just duckdb-tier-c` (timeout <= 30s for medium fixture)
- `just duckdb-tier-c-large` (timeout <= 120s, optional)

## Step 7: Tier D DuckDB-WASM compatibility

Work:
- Add a WASM smoke test that opens a `.duckdb` file and runs:
  - `SELECT COUNT(*) FROM v_points`
  - JSON extraction from `agents.spec`
- Use the latest stable `@duckdb/duckdb-wasm` only.

Tests:
- `just duckdb-tier-d` (timeout <= 15s)

## Step 8: Optional sqllogictest pack

Work:
- Add `tests/duckdb/slt/` with SQL logic tests for multi-connection scenarios.
- Provide a small wrapper to run them from Go or `just`.

Tests:
- `nix develop -c duckdb -slt tests/duckdb/slt/*.slt` (timeout <= 5s per file)

## Step 9: BDD feature file

Work:
- Implement step definitions for `spec/features/duckdb-schema/testing.feature`
  if we decide to run these in godog.

Tests:
- `nix develop -c godog run ./tests/duckdb/godog` (timeout <= 5s)

## Done criteria

- Schema DDL lives in one file and is applied from Go.
- Tier A tests pass on in-memory and on-disk databases.
- Tier B/C/D tests can be run manually via `just` commands.
- Key invariants (value-column correctness, orphans, v_points shape) are
  enforced by tests.
