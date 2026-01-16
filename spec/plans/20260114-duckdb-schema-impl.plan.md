# Plan: Implement DuckDB Schema + Test Suite

Date: 2026-01-14  
Status: DONE

## Goal
Implement the DuckDB measurement schema, ingestion helpers, and Tier A/B/C/D tests
as defined in `spec/features/duckdb-schema/`, including manual `just` commands.

## Non-goals
- Implementing ingestion pipelines or report UI.
- Adding SQL logic tests or godog steps beyond what the feature requires.

## Decisions
- Keep the DDL in `internal/duckdb/schema.sql` and load it from Go.
- Use `github.com/duckdb/duckdb-go/v2` for local testing.
- Use deterministic, seed-based fixture generation for medium/large fixtures,
  stored as fixture definitions under `tests/fixtures/duckdb/`.

## Step 1: Dev environment + dependencies

Work:
- Add DuckDB CLI/lib, a C toolchain, and nodejs to `flake.nix`.
- Add Go module dependency for the DuckDB driver.

Tests:
- None (environment + dependency changes only).

## Step 2: Schema DDL + loader helpers

Work:
- Add `internal/duckdb/schema.sql` from the spec.
- Implement schema loader helpers + base test utilities.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test).

## Step 3: Ingestion helpers + unit tests

Work:
- Add canonical JSON + fingerprint helpers.
- Implement upsert helpers for agents/questions/contexts.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test).

## Step 4: Tier A constraints + invariants

Work:
- Add schema creation, PK/unique, JSON/MAP behavior, value-column invariants,
  and orphan checks.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test).

## Step 5: v_points view contract tests

Work:
- Add fixtures and tests for v_points shape + semantics.

Tests:
- `go test ./internal/duckdb/...` (timeout <= 2s per test).

## Step 6: Tier B fuzz/property tests

Work:
- Add randomized generators for specs and canonicalization stability checks.
- Persist failing seeds to `tests/fixtures/duckdb/fuzz/`.

Tests:
- `just duckdb-tier-b` (timeout <= 5s per test file).

## Step 7: Tier C performance + durability

Work:
- Implement medium fixture loader + core report query timing.
- Add crash-safety smoke test for on-disk DBs.

Tests:
- `just duckdb-tier-c` (timeout <= 30s).
- `just duckdb-tier-c-large` (timeout <= 120s, optional).

## Step 8: Tier D DuckDB-WASM smoke test

Work:
- Add Node-based WASM smoke test script using latest stable `@duckdb/duckdb-wasm`.
- Provide a `just duckdb-tier-d` command.

Tests:
- `just duckdb-tier-d` (timeout <= 15s).

## Done criteria
- Schema DDL lives in one file and is applied from Go.
- Tier A/B/C/D tests are implemented with explicit timeouts.
- Manual `just` commands exist for Tier B/C/D.
- Fixtures live under `tests/fixtures/duckdb/`.
