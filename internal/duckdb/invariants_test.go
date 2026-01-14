package duckdb_test

import (
	"testing"

	"github.com/google/uuid"
)

// TestMeasurementValueInvariant enforces the typed value column invariant.
func TestMeasurementValueInvariant(t *testing.T) {
	db, ctx := openTestDB(t)
	metricID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, 'tokens', 'BIGINT')", metricID)

	runID := uuid.NewString()
	contextID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id, value_bigint) VALUES (?, ?, ?, 42)", runID, contextID, metricID)

	invariantQuery := `SELECT count(*) AS bad_rows
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
);`

	if bad := queryInt(t, ctx, db, invariantQuery); bad != 0 {
		t.Fatalf("expected 0 invalid measurements, got %d", bad)
	}

	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id, sample_index, value_bigint, value_double) VALUES (?, ?, ?, 1, 7, 3.14)", runID, contextID, metricID)
	if bad := queryInt(t, ctx, db, invariantQuery); bad != 1 {
		t.Fatalf("expected 1 invalid measurement, got %d", bad)
	}
}

// TestOrphanChecks validates manual referential integrity queries.
func TestOrphanChecks(t *testing.T) {
	db, ctx := openTestDB(t)
	repoID := uuid.NewString()
	revID := "rev-orphan"
	contextID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, 'orphan-context', ?, ?)", contextID, repoID, revID)

	contextOrphans := queryInt(t, ctx, db, `SELECT count(*) AS orphans
FROM contexts c
LEFT JOIN revisions r
  ON r.repo_id = c.repo_id AND r.rev_id = c.rev_id
WHERE r.rev_id IS NULL;`)
	if contextOrphans != 1 {
		t.Fatalf("expected 1 context orphan, got %d", contextOrphans)
	}

	measurementID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id) VALUES (?, ?, ?)", uuid.NewString(), measurementID, uuid.NewString())
	measurementOrphans := queryInt(t, ctx, db, `SELECT count(*) AS orphans
FROM measurements m
LEFT JOIN contexts c ON c.context_id = m.context_id
WHERE c.context_id IS NULL;`)
	if measurementOrphans != 1 {
		t.Fatalf("expected 1 measurement orphan, got %d", measurementOrphans)
	}
}
