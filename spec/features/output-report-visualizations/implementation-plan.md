# Implementation Plan (v1)

## Step 1: Refactor bootstrap + types

- Split `web/src/main.ts` into report modules (state, ui, duckdb, plots).
- Add Zod schemas and strict TypeScript types for query rows.
- Keep the existing "status pill" behavior working.
- Tests: add Vitest and a first unit test for `formatMetricLabel`.

## Step 2: Metric selector + point view

- Load numeric metrics from `metric_defs`.
- Default to the first metric (alphabetical) and render the points view.
- Build `metric_points` temp view and render `vg.dot`.
- If `v_edges` exists, add `edge_xy` and render `vg.link`.
- Tests: metric selection logic + SQL builder safety.

## Step 3: Candlestick view

- Add view toggle (Points | Candles).
- Build `metric_candles` temp view and render wick + body `ruleX` layers.
- If `v_component_edges` exists, add `component_edge_xy` and render links.
- Tests: table detection + candles SQL builder.

## Step 4: Empty + error states

- When no metrics exist: show empty state and keep chart blank.
- When candles view selected but `v_candles` missing: show warning and blank
  chart (no crash).
- Update details line for warnings and missing-data messages.
- Tests: table detection scenarios and empty state behavior.

## Step 5: BDD scenarios

- Add/adjust `testing.feature` scenarios for the metric selector and view
  toggles.

## Completion

Mark this plan and status file DONE when the UI and tests are complete.
