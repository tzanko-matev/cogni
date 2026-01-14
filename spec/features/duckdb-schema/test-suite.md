# Test Suite Specification (v1)

This test suite is correctness-first and mirrors the memo in
`spec/inbox/duckdb-schema-test.md`. All tiers (A/B/C/D) are included in this
spec and are run manually via `just` commands.

## Test tiers

### Tier A (required)

- Schema creation (DDL runs from scratch).
- Primary key + unique constraint enforcement.
- JSON/MAP semantics for `spec`, `scope`, and `dims`.
- Measurement value-column invariants.
- Orphan checks (manual referential integrity).
- `v_points` view shape + row semantics.

### Tier B (property-based + fuzz, Go only)

- Property-based fuzz for agent/question/context specs.
- Canonicalization collision detection (statistical).
- Seeded, deterministic generators; failing seeds are saved to disk.

Suggested approach (Go standard library only):
- Use `testing/quick` for randomized generators.
- Wrap the RNG with a logged seed and write failing inputs to
  `tests/fixtures/duckdb/fuzz/seed-<seed>.json`.

### Tier C (performance + durability)

- Large-scale performance + concurrency using on-disk DBs.
- Crash safety and durability on-disk.
- Performance target: 10k commits with 10 metrics per commit must support
  core report queries in <5s on a developer laptop.

Core report queries to benchmark:
- tokens over time (`v_points` filter by metric + status)
- latest value per context
- compare two runs for same context/metric

Measure with `EXPLAIN ANALYZE` or timed Go queries and record results.

### Tier D (compatibility: DuckDB-WASM, latest stable)

- Verify the latest stable DuckDB version can open the generated `.duckdb` file
  in the browser (DuckDB-WASM).
- Ensure the `v_points` view is readable and returns expected rows.
- JSON extraction (`->`, `->>`) works in WASM for agent/question specs.

## Test harness (Go)

Use Go tests with explicit timeouts. The core helpers should live in a dedicated
package (example: `internal/duckdb/testing`).

### Minimal helper: open DB with timeout

```go
func openDuckDB(t *testing.T, dsn string) *sql.DB {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	conn, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		t.Fatalf("ping duckdb: %v", err)
	}
	return conn
}
```

### Running the schema DDL

```go
func applySchema(t *testing.T, db *sql.DB, ddl string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	t.Cleanup(cancel)
	if _, err := db.ExecContext(ctx, ddl); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}
```

## Tier B example: canonicalization stability (Go)

```go
func TestCanonicalJSONStableKeyOrdering(t *testing.T) {
	f := func(seed int64) bool {
		rng := rand.New(rand.NewSource(seed))
		spec := randomSpec(rng)
		a, err := CanonicalJSON(spec)
		if err != nil {
			return false
		}
		// Shuffle map key order by re-marshaling through map iteration.
		spec2 := rehydrateMap(spec)
		b, err := CanonicalJSON(spec2)
		if err != nil {
			return false
		}
		return bytes.Equal(a, b)
	}
	cfg := &quick.Config{MaxCount: 200, Rand: rand.New(rand.NewSource(42))}
	if err := quick.Check(f, cfg); err != nil {
		t.Fatalf("canonicalization not stable: %v", err)
	}
}
```

Notes:
- Log and persist seeds on failure to `tests/fixtures/duckdb/fuzz/`.
- Keep test runtime under 2s.

## Core invariant queries

### Measurement type invariant

Exactly one value column should be set when `status='ok'`.

```sql
SELECT count(*) AS bad_rows
FROM measurements m
JOIN metric_defs md ON md.metric_id = m.metric_id
WHERE m.status = 'ok'
AND (
  (md.physical_type = 'DOUBLE' AND m.value_double IS NULL) OR
  (md.physical_type = 'BIGINT' AND m.value_bigint IS NULL) OR
  (md.physical_type = 'BOOLEAN' AND m.value_bool IS NULL) OR
  (md.physical_type = 'VARCHAR' AND m.value_varchar IS NULL) OR
  (md.physical_type = 'JSON' AND m.value_json IS NULL) OR
  (md.physical_type = 'BLOB' AND m.value_blob IS NULL)
  OR
  (m.value_double IS NOT NULL)::INT +
  (m.value_bigint IS NOT NULL)::INT +
  (m.value_bool IS NOT NULL)::INT +
  (m.value_varchar IS NOT NULL)::INT +
  (m.value_json IS NOT NULL)::INT +
  (m.value_blob IS NOT NULL)::INT <> 1
);
```

### Orphan checks (examples)

```sql
-- Contexts must reference existing revisions.
SELECT count(*) AS orphans
FROM contexts c
LEFT JOIN revisions r
  ON r.repo_id = c.repo_id AND r.rev_id = c.rev_id
WHERE r.rev_id IS NULL;

-- Measurements must reference existing contexts.
SELECT count(*) AS orphans
FROM measurements m
LEFT JOIN contexts c ON c.context_id = m.context_id
WHERE c.context_id IS NULL;
```

## v_points view contract tests

- `value` is DOUBLE for DOUBLE/BIGINT metrics and NULL otherwise.
- `ts` is taken from `revisions.ts_utc`.
- Filtering by `metric` and `status` returns expected rows.

Example assertion query:

```sql
SELECT
  COUNT(*) AS rows
FROM v_points
WHERE metric = 'tokens'
  AND status = 'ok'
  AND ts IS NOT NULL;
```

## JSON + MAP behavior tests

- JSON parse errors should fail inserts.
- JSON extraction with `->` and `->>` works for agent specs.
- MAP supports missing keys and varying schemas.

Example MAP extraction:

```sql
SELECT dims['benchmark'] AS benchmark
FROM contexts
WHERE dims IS NOT NULL;
```

## Tier D example: DuckDB-WASM smoke test (TypeScript)

```ts
import * as duckdb from "@duckdb/duckdb-wasm";

export async function runWasmSmokeTest(dbPath: string) {
  const bundles = duckdb.getJsDelivrBundles();
  const bundle = await duckdb.selectBundle(bundles);
  const worker = new Worker(bundle.mainWorker);
  const logger = new duckdb.ConsoleLogger();
  const db = new duckdb.AsyncDuckDB(logger, worker);
  await db.instantiate(bundle.mainModule, bundle.pthreadWorker);
  await db.registerFileURL("cogni.duckdb", dbPath);
  const conn = await db.connect();
  await conn.query("SELECT COUNT(*) FROM v_points");
  await conn.query("SELECT spec->>'$.model' FROM agents LIMIT 1");
  await conn.close();
  await db.terminate();
}
```

Notes:
- The smoke test only validates open + query, not performance.
- Use the latest stable DuckDB-WASM package.

## Fixture strategy

- Tiny fixture (correctness): 20 revisions, 2 agents, 2 questions, 3 metrics,
  include at least one `status='error'` measurement for invariant testing.
- Medium fixture (performance target): 10,000 revisions, 1 agent, 1 question,
  10 metrics per revision (100,000 measurements). This is the baseline for the
  <5s query requirement.
- Large fixture (stress): 100,000 revisions, 1 agent, 1 question, 10 metrics
  per revision (1,000,000 measurements). Optional manual run.
- Keep fixtures under version control in `tests/fixtures/duckdb/`.

Next: `implementation-plan.md`
