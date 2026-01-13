# Plan: DuckDB Schema + Test Suite Spec Pack

Date: 2026-01-14  
Status: DONE

## Goal
Create a junior-developer-friendly documentation pack in `spec/features/` that
specifies the DuckDB measurement schema and its test suite based on the inbox
memos for DuckDB research and testing.

## Non-goals
- Implementing the schema in code.
- Writing the tests or DB migrations.
- Implementing ingestion or report features.

## Decisions
- The spec pack lives in `spec/features/duckdb-schema/`.
- Include practical SQL and Go snippets for the tricky pieces
  (canonical JSON fingerprinting, upserts, invariant checks).
- Call out any dev-environment changes needed in `flake.nix`.

## Step 1: Create feature pack scaffolding + overview

Work:
- Create `spec/features/duckdb-schema/` folder.
- Add `overview.md` with goals, non-goals, read order, and context.

Tests:
- None (documentation-only change).

## Step 2: Document schema + ingestion invariants

Work:
- Add schema DDL doc with table-by-table rationale.
- Include `v_points` view and derived metrics table.
- Add ingestion notes: canonicalization, hashing, idempotent upserts.

Tests:
- None (documentation-only change).

## Step 3: Document test suite + BDD feature

Work:
- Add `test-suite.md` describing tiers, invariants, and fixtures.
- Add `testing.feature` with core invariants in Gherkin.

Tests:
- None (documentation-only change).

## Step 4: Add implementation plan + dev environment notes

Work:
- Add `implementation-plan.md` with steps and explicit test commands.
- Document any required updates to `flake.nix` for DuckDB tooling.

Tests:
- None (documentation-only change).

## Done criteria
- The spec pack exists and references the DuckDB memos.
- All files are readable for a junior developer and include key snippets.
- Plan and status docs are marked DONE.
