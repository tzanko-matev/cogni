# DuckDB Schema + Test Suite Feature Spec (v1)

Audience: junior Go developer. This spec is self-contained. Follow the files in
this folder in the order below.

## Read order

1) `overview.md` (this file)
2) `schema.md`
3) `ingestion.md`
4) `test-suite.md`
5) `implementation-plan.md`
6) `testing.feature`

## Context

We are preparing two upcoming features:
1) Cogni writes measurements to DuckDB files.
2) `cogni serve` opens reports and ships a DuckDB file to the browser for
   interactive exploration (vgplot + DuckDB WASM).

This spec only covers the DuckDB **schema** and its **test suite**. It is based
on:
- `spec/inbox/duckdb-research.md`
- `spec/inbox/duckdb-schema-test.md`

## Goals

- Implement the DuckDB measurement schema described in the inbox memos.
- Provide a stable `v_points` view for plot-ready data.
- Define a correctness-first test suite that validates schema, invariants, and
  view contracts across all tiers (A, B, C, D).
- Keep the schema append-only for measurements and idempotent for dimension
  tables (agents/questions/contexts).

## Non-goals (v1)

- Implementing ingestion pipelines or the reporting UI.
- Computing commit graph edges, connected components, or candlestick grouping
  in SQL (this stays client-side).
- Adding an `agent_calls` trace table.
- Adding foreign key constraints (we enforce integrity via tests instead).

## Decisions (source of truth)

- Use a normalized schema: `agents` and `questions` store JSON specs; `contexts`
  ties repo+revision+agent+question+extra dims together.
- Use typed value columns in `measurements` and enforce correctness via tests.
- Store derived metrics as SQL definitions in `derived_metric_defs` (not
  materialized by default).
- Keep schema DDL in one place and make it easy to load from Go tests.
- Tier B uses Go tests only (no external property-based tools).
- Tier C performance target: 10k commits, 10 metrics per commit, <5s query
  latency for core reporting queries.
- Tier D prioritizes DuckDB-WASM compatibility with the latest stable release.
- Tests are manual-only (run via `just` commands); no CI/CD requirement yet.

## Deliverables in implementation

- `internal/duckdb/schema.sql` (or equivalent) containing the DDL.
- Go helpers to load the schema and to upsert dimension tables.
- Tests for:
  - primary key + unique constraint enforcement
  - value column invariants
  - orphan checks
  - `v_points` view shape and semantics
  - All remaining tests in Tier A,B,C,D from the test suite design

## Development environment notes

If we use the Go DuckDB driver (`github.com/marcboeker/go-duckdb`) we may need:
- `duckdb` (CLI and library) in the Nix dev shell
- a C toolchain for CGO (`clang`/`gcc` + `pkg-config`)

See `implementation-plan.md` for the exact proposal.

Next: `schema.md`
