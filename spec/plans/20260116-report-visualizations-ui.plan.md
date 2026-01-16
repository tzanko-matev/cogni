# Plan: Report visualizations UI (points + candles)

Date: 2026-01-16
Owner: Codex

## Goal

Implement the report visualizations UI in `web/` with:
- Metric selector
- View toggle (Points/Candles)
- Bucket size selector (Day/Week/Month)
- Client-side graph + component computation from `revision_parents`

## Constraints

- Keep files under 200 lines.
- Add docstrings to all functions.
- Default bucket size resets to Day on each load (no persistence).

## Steps

### Step 1: Project scaffolding + types

- Add TypeScript modules for report UI (`web/src/report/*`).
- Add Zod schemas and shared types.
- Wire new bootstrap entrypoint from `web/src/main.ts`.
- Tests:
  - Add Vitest config and a small unit test for `formatMetricLabel`.

### Step 2: DuckDB integration + metric selection

- Implement DuckDB init, attach, metric list query, and `metric_points` temp view.
- Implement selector wiring and default metric selection.
- Tests:
  - Metric selection logic (numeric-only, default selection, preserve selection).
  - SQL escaping helper.

### Step 3: Graph computation + points view

- Implement client-side minimal edge computation from `revision_parents`.
- Load `edge_xy` temp table and render points view with optional links.
- Tests:
  - `computeMinimalEdges` stops at first measured ancestor.

### Step 4: Components + candles view

- Implement bucket assignment (day/week/month).
- Compute per-bucket components and candle aggregates.
- Load `metric_candles` + `component_edge_xy` temp tables.
- Add bucket-size control and re-render on change.
- Tests:
  - `bucketKey` outputs for day/week/month.
  - `computeComponents`, `computeCandles`, `computeComponentLinks`.

### Step 5: UI polish + states

- Add warnings/empty states for missing `revision_parents` and no metrics.
- Ensure controls disable appropriately and details line updates.
- Tests:
  - Table detection state changes.

## Completion Criteria

- UI renders points and candles from the report DB.
- Bucket size control recomputes candles.
- All unit tests pass (`vitest run`).
- Plan and status files marked DONE.
