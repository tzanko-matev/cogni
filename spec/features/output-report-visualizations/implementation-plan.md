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
- If `revision_parents` exists, compute minimal edges client-side, load
  `edge_xy`, and render `vg.link`.
- Tests: metric selection logic + SQL builder safety + edge computation.

## Step 3: Candlestick view

- Add view toggle (Points | Candles).
- Compute per-bucket components client-side and load `metric_candles` temp
  table; render wick + body `ruleX` layers.
- Compute cross-bucket component links and load `component_edge_xy`.
- Tests: component bucketing + candle computation + link dedupe.

## Step 4: Empty + error states

- When no metrics exist: show empty state and keep chart blank.
- When `revision_parents` is missing: disable edges and Candles view; show a
  warning and keep the chart stable (no crash).
- Update details line for warnings and missing-data messages.
- Tests: table detection scenarios and empty state behavior.

## Step 5: BDD scenarios

- Add/adjust `testing.feature` scenarios for the metric selector and view
  toggles.

## Completion

Mark this plan and status file DONE when the UI and tests are complete.
