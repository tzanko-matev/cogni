# Plan: Expand DuckDB Test Tiers (B/C/D)

Date: 2026-01-14  
Status: DONE

## Goal
Expand the DuckDB schema spec pack to include complete Tier B, C, and D testing
requirements, including manual test commands and concrete fixture sizes.

## Non-goals
- Implementing any tests or code.
- Adding CI/CD configuration.

## Decisions
- Tier B uses Go tests only (no external property-based tools).
- Tier C performance target: 10k commits, 10 metrics per commit, <5s query latency.
- Tier D focuses on DuckDB-WASM with latest stable DuckDB only.
- All tests are run manually via `just` commands.
- Fixture sizes will be specified in the spec.

## Step 1: Update overview + test suite docs

Work:
- Update `overview.md` to reflect full tier coverage and manual test mode.
- Expand `test-suite.md` with detailed Tier B/C/D requirements.

Tests:
- None (documentation-only change).

## Step 2: Update implementation plan

Work:
- Add explicit manual `just` commands for Tier B/C/D execution.
- Add fixture sizes and performance thresholds.

Tests:
- None (documentation-only change).

## Done criteria
- Spec pack includes all tiers with concrete requirements.
- Manual test commands and fixture sizes are documented.
- Plan and status docs are marked DONE.
