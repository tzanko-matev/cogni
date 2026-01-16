# Status: Report visualizations UI (points + candles)

Date: 2026-01-16
Plan: `spec/plans/20260116-report-visualizations-ui.plan.md`

## Scope

Implement the report visualizations UI in `web/` with metric selection, view
toggle, bucket size control, and client-side graph/component computation.

## Relevant specs

- `spec/features/output-report-visualizations/overview.md`
- `spec/features/output-report-visualizations/ui.md`
- `spec/features/output-report-visualizations/integration.md`
- `spec/features/output-report-visualizations/test-suite.md`
- `spec/features/output-report-visualizations/testing.feature`

## Relevant files

- `web/src/main.ts`
- `web/src/style.css`
- `web/src/report/*`
- `web/package.json`
- `web/tsconfig.json`

## Progress

- Status: IN_PROGRESS
- Last updated: 2026-01-16
- Completed:
  - Plan/status files created.
  - Step 1 scaffolding: report modules, Zod types, Vitest config, and a format label test.
  - Step 2 metric selection: DuckDB queries, metric_points view, and selector wiring.
  - Step 3 points view edges: client-side minimal edges and edge_xy temp table.
  - Step 4 candlesticks: bucketed components, candles, and component links.
- In progress:
  - Step 5: UI polish + empty/error states.

## Notes

- Bucket size should default to Day on each load (no persistence).
- Tests not run yet (vitest).
