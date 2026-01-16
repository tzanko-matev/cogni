# Test Suite (v1)

All automated tests must set explicit timeouts (<= 1s unless noted).

## Unit tests (TypeScript)

Add a lightweight test runner (Vitest recommended) under `web/`.

### 1) Metric selection logic

- Filters to numeric metrics only.
- Chooses the first numeric metric alphabetically when no previous selection.
- Preserves the previous selection if it is still available.

Timeout: 1s.

### 2) SQL builder safety

- `escapeSqlString` properly quotes metric names containing `'`.
- `buildMetricPointsViewSQL(metric)` includes status and value filters.

Timeout: 1s.

### 3) Table detection

- `detectTables` returns `true` for `revision_parents` when present.
- Missing `revision_parents` disables edges and Candles view and triggers a
  warning state.

Timeout: 1s.

### 4) Label formatting

- `formatMetricLabel` includes the unit when provided ("Tokens (count)").
- Falls back to metric name alone when unit is empty.

Timeout: 1s.

### 5) Graph + component computation

- `bucketKey` produces correct UTC day/week/month keys.
- `computeMinimalEdges` stops at the first measured ancestor.
- `computeComponents` groups nodes only when edges are within the same bucket.
- `computeCandles` returns correct open/close/high/low for a component.
- `computeComponentLinks` deduplicates links across buckets.

Timeout: 1s.

## Integration tests (optional, manual-only for v1)

- Load a tiny DuckDB file in the browser and confirm:
  - metric selector populates
  - point view renders dots
  - candle view renders wick + body

If automation is added later, ensure each test has a 2s timeout.

## BDD scenarios

Behavior is described in `spec/features/output-report-visualizations/testing.feature`.
