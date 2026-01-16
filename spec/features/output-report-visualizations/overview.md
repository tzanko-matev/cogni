# Report Visualizations (Points + Candles) (v1)

Audience: junior TypeScript developer. This spec is self-contained. Read the
files in order.

## Read order

1) `overview.md` (this file)
2) `ui.md`
3) `integration.md`
4) `test-suite.md`
5) `implementation-plan.md`
6) `testing.feature`

## Context

We already serve a DuckDB report file via `cogni serve` and render a placeholder
vgplot chart in the browser. The next step is to ship the **real** reporting
visualizations described in:

- `spec/inbox/vgplot-research.md`
- `spec/inbox/duckdb-research.md`

## Goal

Add a metric picker and two view modes to the report UI:

1) **Point view**: measured commits as dots, with minimal ancestor edges when
   available.
2) **Candlestick view**: per-day connected-component candles, with optional thin
   links between components.

The user must be able to choose a metric and swap between these two views.

## Scope

- Browser UI under `web/` only (TypeScript + Vite).
- Uses DuckDB-WASM for data access and vgplot for rendering.
- Uses the DuckDB report file served at `/data/db.duckdb`.
- Supports **numeric metrics only** (`physical_type` DOUBLE or BIGINT).

## Non-goals (v1)

- Computing git/jj graph algorithms in the browser (edges, components).
- Changing the DuckDB schema or report generation pipeline.
- Advanced interactions (brush/zoom, pan, multi-metric overlays).
- Cross-repo selection (assume a single repo in the report file).

## Decisions (source of truth)

- Use vgplot marks: `dot` + `link` for point view; `ruleX` + optional `link` for
  candle view.
- Populate the metric selector from `metric_defs` and filter to numeric metrics.
- Only plot rows with `status = 'ok'` and `value IS NOT NULL`.
- Build **temporary views** in DuckDB per selected metric; do not materialize
  new tables on disk.
- If required tables for edges or candles are missing, show a clear warning and
  degrade gracefully (dots-only or empty-state) instead of failing.

## Deliverables

- Refactored `web/src` UI with a metric selector + view toggle.
- Point view rendering using `v_points` and optional `v_edges`.
- Candlestick view rendering using `v_candles` and optional
  `v_component_edges`.
- Tests for selector logic, view SQL generation, and error/empty states.

Next: `ui.md`
