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
- `web/src/report/graph.ts` (client-side graph + component logic)
- `web/src/report/types.ts` (types + Zod schemas)

## Data contract (DuckDB)

The UI reads from the report DuckDB file, attached as `cogni`.

### Required tables/views

`cogni.main.v_points` (already in schema)

Columns used by UI:
- `repo_id` (UUID)
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

`cogni.main.runs`

Columns used by UI:
- `run_id` (UUID)
- `collected_at` (TIMESTAMP)

### Optional table (recommended for graph computations)

`cogni.main.revision_parents`

- Purpose: parent adjacency for ancestry traversal.
- Columns:
  - `repo_id` (UUID)
  - `child_rev_id` (VARCHAR)
  - `parent_rev_id` (VARCHAR)

Notes:
- If `revision_parents` is missing, we cannot compute edges or components; the
  UI must fall back to a dots-only chart and disable the Candles view.
- If multiple repos exist, pick the repo_id that appears in `metric_points`
  (v1 assumes a single repo).

## DuckDB access

### Metric list

Query numeric metrics for the selector:

```sql
SELECT name, description, unit, physical_type
FROM cogni.main.metric_defs
WHERE physical_type IN ('DOUBLE','BIGINT')
ORDER BY name;
```

Note: `$metric` in this document is a string literal placeholder. Use a small
SQL-escape helper (e.g., replace `'` with `''`) before interpolation to avoid
SQL injection and syntax errors.

### Latest-per-commit points

To avoid multiple points per commit when multiple runs exist, build a temp view
for the selected metric using the latest `runs.collected_at` timestamp:

```sql
CREATE OR REPLACE TEMP VIEW metric_points AS
SELECT
  repo_id,
  rev_id,
  ts,
  arg_max(value, run_ts) AS value
FROM (
  SELECT v.repo_id, v.rev_id, v.ts, v.value, r.collected_at AS run_ts
  FROM cogni.main.v_points v
  JOIN cogni.main.runs r ON r.run_id = v.run_id
  WHERE v.metric = $metric
    AND v.status = 'ok'
    AND v.value IS NOT NULL
) grouped
GROUP BY repo_id, rev_id, ts;
```

Then load the rows into memory for client-side graph computation:

```sql
SELECT repo_id, rev_id, ts, value
FROM metric_points
ORDER BY ts;
```

### Parent edges

If `revision_parents` exists, load parent edges for the same repo:

```sql
SELECT child_rev_id, parent_rev_id
FROM cogni.main.revision_parents
WHERE repo_id = $repo_id;
```

## Client-side graph computation

We compute **minimal edges** and **per-bucket connected components** in the
browser so grouping can change (day/week/month) without regenerating the report.

### Inputs

- `metricPoints`: array of `{ revId, ts, value }`.
- `parents`: map `childRevId -> parentRevId[]` from `revision_parents`.
- `bucketSize`: one of `day | week | month` (v1: default to `day`).

### Step 1: Index points

```ts
const pointByRev = new Map(revId -> { ts, value });
const measured = new Set(revId);
```

### Step 2: Minimal ancestor edges (measured graph)

For each measured revision, walk **upwards** through parents until you reach
another measured revision. Those first measured ancestors become the minimal
edges into this node.

Key rule: **do not traverse past a measured ancestor**; that is the transitive
reduction over measured nodes.

Pseudo:

```ts
function nearestMeasuredAncestors(revId): Set<revId> {
  const result = new Set();
  const queue = [...(parents.get(revId) ?? [])];
  const seen = new Set(queue);
  while (queue.length > 0) {
    const current = queue.shift();
    if (measured.has(current)) {
      result.add(current);
      continue; // stop at first measured ancestor on this path
    }
    for (const p of parents.get(current) ?? []) {
      if (!seen.has(p)) {
        seen.add(p);
        queue.push(p);
      }
    }
  }
  return result;
}

const edges = [];
for (const revId of measured) {
  for (const parent of nearestMeasuredAncestors(revId)) {
    edges.push({ parent, child: revId });
  }
}
```

Optimization: memoize `nearestMeasuredAncestors` per revision and reuse across
nodes to avoid repeated traversal in large graphs.

### Step 3: Bucket assignment

Bucket by UTC time so grouping is stable across time zones:

```ts
function bucketKey(ts: Date, bucketSize: 'day'|'week'|'month'): string {
  // day: YYYY-MM-DD (UTC)
  // week: YYYY-Www (ISO week, UTC)
  // month: YYYY-MM (UTC)
}
```

Store `bucketByRev` for quick lookup.

### Step 4: Connected components per bucket

For each bucket, compute connected components using the **undirected** version
of the measured graph restricted to that bucket.

- Include an edge if both endpoints are in the same bucket.
- Use union-find (disjoint set) or DFS/BFS on the bucket-local graph.

Assign component ids as `${bucketKey}:${index}` to keep them stable within the
bucket.

### Step 5: Candles + component links

For each component:

- `open`: value at earliest `ts` within the component.
- `close`: value at latest `ts` within the component.
- `low/high`: min/max value in the component.
- `x_ts`: representative timestamp (use midpoint of min/max ts or mean).

For thin links:

- For every **edge** whose endpoints are in **different buckets**, add a link
  between their component ids.
- Deduplicate links with a set of `(fromComponentId,toComponentId)`.

### Outputs (arrays)

- `edgeXY`: `{ x1, y1, x2, y2 }` for point view links.
- `candles`: `{ bucket, componentId, x, open, close, low, high }`.
- `componentEdgeXY`: `{ x1, y1, x2, y2 }` for candle view links.

## Loading derived data into DuckDB

vgplot queries tables, so load the computed arrays into **TEMP** tables.

Recommended approach: use DuckDB's Arrow insertion (fast, avoids huge SQL
strings). If Arrow is too heavy for v1, fall back to `INSERT` loops for small
datasets.

Create temp tables:

```sql
CREATE OR REPLACE TEMP TABLE edge_xy (x1 TIMESTAMP, y1 DOUBLE, x2 TIMESTAMP, y2 DOUBLE);
CREATE OR REPLACE TEMP TABLE metric_candles (
  bucket VARCHAR,
  component_id VARCHAR,
  x TIMESTAMP,
  open DOUBLE,
  close DOUBLE,
  low DOUBLE,
  high DOUBLE
);
CREATE OR REPLACE TEMP TABLE component_edge_xy (x1 TIMESTAMP, y1 DOUBLE, x2 TIMESTAMP, y2 DOUBLE);
```

Then insert rows (Arrow preferred):

```ts
await conn.insertArrowTable(edgeTable, { name: 'edge_xy' });
await conn.insertArrowTable(candleTable, { name: 'metric_candles' });
await conn.insertArrowTable(componentEdgeTable, { name: 'component_edge_xy' });
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

- On startup, query `information_schema.tables` to detect whether
  `revision_parents` exists.
- If missing, disable edges and Candles view; show a warning in the details
  line (do not throw).
- If no numeric metrics exist, show an empty state and keep the chart blank.

## Runtime validation

- Add `zod` to `web/package.json` and define schemas for:
  - metric defs rows
  - metric points rows
  - candle rows
  - edge rows
- Parse query results with Zod before using them in chart code.
- Avoid `any` and keep all exported functions explicitly typed.

Next: `test-suite.md`
