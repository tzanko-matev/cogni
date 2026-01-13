# Test Suite Specification (v1)

This test suite is correctness-first and mirrors the memo in
`spec/inbox/duckdb-schema-test.md`. Start with Tier A tests; Tier B/C can be
added later when the schema is stable.

## Test tiers

### Tier A (required for v1)

- Schema creation (DDL runs from scratch).
- Primary key + unique constraint enforcement.
- JSON/MAP semantics for `spec`, `scope`, and `dims`.
- Measurement value-column invariants.
- Orphan checks (manual referential integrity).
- `v_points` view shape + row semantics.

### Tier B (later)

- Property-based fuzz for agent/question/context specs.
- Canonicalization collision detection (statistical).

### Tier C (later)

- Large-scale performance + concurrency.
- Crash safety and durability on-disk.

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

## Fixture strategy

- Tiny fixture: ~10 revisions, 2 agents, 2 questions, multiple metrics.
- Medium fixture: ~10k revisions, used for `v_points` queries.
- Keep fixtures under version control in `tests/fixtures/duckdb/`.

Next: `implementation-plan.md`
