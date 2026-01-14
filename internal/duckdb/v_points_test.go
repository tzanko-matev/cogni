package duckdb_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestVPointsViewContract validates v_points value and timestamp semantics.
func TestVPointsViewContract(t *testing.T) {
	db, ctx := openTestDB(t)
	repoID := uuid.NewString()
	revID := "rev-1"
	revTime := time.Date(2026, 1, 14, 12, 0, 0, 0, time.UTC)
	execSQL(t, ctx, db, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, 'repo', 'git')", repoID)
	execSQL(t, ctx, db, "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)", repoID, revID, revTime)

	tokensMetricID := uuid.NewString()
	latencyMetricID := uuid.NewString()
	jsonMetricID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, 'tokens', 'BIGINT')", tokensMetricID)
	execSQL(t, ctx, db, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, 'latency', 'DOUBLE')", latencyMetricID)
	execSQL(t, ctx, db, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, 'metadata', 'JSON')", jsonMetricID)

	runID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')", runID, repoID, revTime)

	contextID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, 'context-1', ?, ?)", contextID, repoID, revID)

	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id, value_bigint) VALUES (?, ?, ?, 123)", runID, contextID, tokensMetricID)
	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id, value_double) VALUES (?, ?, ?, 4.2)", runID, contextID, latencyMetricID)
	execSQL(t, ctx, db, "INSERT INTO measurements (run_id, context_id, metric_id, value_json) VALUES (?, ?, ?, '{\"ok\":true}')", runID, contextID, jsonMetricID)

	var ts time.Time
	var value sql.NullFloat64
	if err := db.QueryRowContext(ctx, "SELECT ts, value FROM v_points WHERE metric = 'tokens'").Scan(&ts, &value); err != nil {
		t.Fatalf("select v_points tokens: %v", err)
	}
	if !value.Valid || value.Float64 != 123 {
		t.Fatalf("expected tokens value 123, got %v", value.Float64)
	}
	if !ts.Equal(revTime) {
		t.Fatalf("expected ts %v, got %v", revTime, ts)
	}

	var jsonValue sql.NullFloat64
	if err := db.QueryRowContext(ctx, "SELECT value FROM v_points WHERE metric = 'metadata'").Scan(&jsonValue); err != nil {
		t.Fatalf("select v_points metadata: %v", err)
	}
	if jsonValue.Valid {
		t.Fatalf("expected metadata value to be NULL")
	}
}
