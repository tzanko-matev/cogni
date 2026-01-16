# Integration Design (v1)

This document explains how to wire the visualization UI into the existing
`web/` app while keeping responsibilities clean and testable.

## Target files (suggested)

Keep files under 200 lines. Use JSDoc docstrings for every exported function.
Add a barrel file when introducing a folder.

Suggested structure:

- `web/src/main.ts` (bootstrap only)
- `web/src/report/index.ts` (compose UI)
- `web/src/report/state.ts` (state + reducers)
- `web/src/report/ui.ts` (DOM building + event wiring)
- `web/src/report/duckdb.ts` (DuckDB init + query helpers)
- `web/src/report/plots/points.ts`
- `web/src/report/plots/candles.ts`
- `web/src/report/sql.ts` (SQL builders + escaping)
- `web/src/report/types.ts` (types + Zod schemas)

## Data contract (DuckDB)

The UI reads from the report DuckDB file, attached as `cogni`.

### Existing table/view (required)

`cogni.main.v_points` (already in schema)

Columns used by UI:
- `rev_id` (VARCHAR)
- `ts` (TIMESTAMP)
- `metric` (VARCHAR)
- `value` (DOUBLE)
- `status` (VARCHAR)
- `run_id` (UUID)

`cogni.main.metric_defs`

Columns used by UI:
- `name` (VARCHAR)
- `description` (VARCHAR)
- `unit` (VARCHAR)
- `physical_type` (VARCHAR)

### New views (preferred; produced by report generation)

`cogni.main.v_edges`

- Purpose: minimal ancestor edges between **measured** commits, per metric.
- Columns:
  - `repo_id` (UUID)
  - `metric` (VARCHAR)
  - `parent_rev_id` (VARCHAR)
  - `child_rev_id` (VARCHAR)

`cogni.main.v_candles`

- Purpose: per-day connected-component OHLC values, per metric.
- Columns:
  - `repo_id` (UUID)
  - `metric` (VARCHAR)
  - `day` (DATE)            -- UTC day bucket
  - `component_id` (VARCHAR)
  - `x_ts` (TIMESTAMP)      -- representative timestamp for plotting
  - `open` (DOUBLE)
  - `close` (DOUBLE)
  - `low` (DOUBLE)
  - `high` (DOUBLE)

`cogni.main.v_component_edges` (optional)

- Purpose: thin links between consecutive components across days.
- Columns:
  - `repo_id` (UUID)
  - `metric` (VARCHAR)
  - `from_day` (DATE)
  - `from_component_id` (VARCHAR)
  - `to_day` (DATE)
  - `to_component_id` (VARCHAR)

Notes:
- These views are **produced outside** the UI (graph algorithms are not done
  in SQL or in the browser for v1).
- `component_id` is only unique **within a day**; joins must include `day`.

## DuckDB access

### Metric list

Query numeric metrics for the selector:

```sql
SELECT name, description, unit, physical_type
FROM cogni.main.metric_defs
WHERE physical_type IN ('DOUBLE','BIGINT')
ORDER BY name;
```

### Latest-per-commit points

To avoid multiple points per commit when multiple runs exist, build a temp view
for the selected metric using the latest `runs.collected_at` timestamp:

```sql
CREATE OR REPLACE TEMP VIEW metric_points AS
SELECT
  rev_id,
  ts,
  arg_max(value, run_ts) AS value
FROM (
  SELECT v.rev_id, v.ts, v.value, r.collected_at AS run_ts
  FROM cogni.main.v_points v
  JOIN cogni.main.runs r ON r.run_id = v.run_id
  WHERE v.metric = $metric
    AND v.status = 'ok'
    AND v.value IS NOT NULL
) grouped
GROUP BY rev_id, ts;
```

### Edge coordinates (point view)

If `v_edges` exists, build `edge_xy`:

```sql
CREATE OR REPLACE TEMP VIEW edge_xy AS
SELECT
  e.parent_rev_id,
  e.child_rev_id,
  p.ts    AS x1,
  p.value AS y1,
  c.ts    AS x2,
  c.value AS y2
FROM cogni.main.v_edges e
JOIN metric_points p ON p.rev_id = e.parent_rev_id
JOIN metric_points c ON c.rev_id = e.child_rev_id
WHERE e.metric = $metric;
```

### Candles + component links

```sql
CREATE OR REPLACE TEMP VIEW metric_candles AS
SELECT day, component_id, x_ts AS x, open, close, low, high
FROM cogni.main.v_candles
WHERE metric = $metric;
```

If `v_component_edges` exists, build `component_edge_xy`:

```sql
CREATE OR REPLACE TEMP VIEW component_edge_xy AS
SELECT
  f.x AS x1,
  (f.open + f.close) / 2 AS y1,
  t.x AS x2,
  (t.open + t.close) / 2 AS y2
FROM cogni.main.v_component_edges ce
JOIN metric_candles f
  ON f.day = ce.from_day AND f.component_id = ce.from_component_id
JOIN metric_candles t
  ON t.day = ce.to_day AND t.component_id = ce.to_component_id
WHERE ce.metric = $metric;
```

## vgplot rendering

### Point view plot

```ts
vg.plot(
  edgeAvailable ? vg.link(vg.from("edge_xy"), { x1: "x1", y1: "y1", x2: "x2", y2: "y2", strokeOpacity: 0.25 }) : null,
  vg.dot(vg.from("metric_points"), { x: "ts", y: "value", r: 3, fill: "#ff7a59", stroke: "#1d1a16" }),
  vg.xLabel("Commit time"),
  vg.yLabel(metricLabel)
)
```

### Candlestick plot

```ts
vg.plot(
  linkAvailable ? vg.link(vg.from("component_edge_xy"), { x1: "x1", y1: "y1", x2: "x2", y2: "y2", strokeOpacity: 0.35 }) : null,
  vg.ruleX(vg.from("metric_candles"), { x: "x", y1: "low",  y2: "high", stroke: "component_id", strokeOpacity: 0.7 }),
  vg.ruleX(vg.from("metric_candles"), { x: "x", y1: "open", y2: "close", stroke: "component_id", strokeWidth: 4 }),
  vg.xLabel("Commit time"),
  vg.yLabel(metricLabel)
)
```

## Error handling + table detection

- On startup, query `information_schema.tables` to detect whether `v_edges`,
  `v_candles`, and `v_component_edges` exist.
- Store availability flags in UI state and surface warnings in the details
  line (do not throw).
- If `v_candles` is missing, the Candles view should show an empty state rather
  than failing.

## Runtime validation

- Add `zod` to `web/package.json` and define schemas for:
  - metric defs rows
  - metric points rows
  - candle rows
- Parse query results with Zod before using them in chart code.
- Avoid `any` and keep all exported functions explicitly typed.

Next: `test-suite.md`
