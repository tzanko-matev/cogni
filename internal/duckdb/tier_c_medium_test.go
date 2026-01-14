//go:build duckdbtierc

package duckdb_test

import (
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/duckdb/testing"
	"cogni/internal/testutil"
)

// TestTierCMediumPerformance loads the medium fixture and checks query latency.
func TestTierCMediumPerformance(t *testing.T) {
	ctx := testutil.Context(t, 30*time.Second)
	dbPath := filepath.Join(t.TempDir(), "medium.duckdb")
	db := duckdbtesting.Open(t, dbPath)
	duckdbtesting.ApplySchema(t, db)

	cfg, err := loadFixtureConfig("medium")
	if err != nil {
		t.Fatalf("load fixture config: %v", err)
	}
	data, err := loadFixture(ctx, db, cfg)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	queries := []struct {
		name  string
		query string
		args  []interface{}
	}{
		{
			name:  "tokens_over_time",
			query: "SELECT ts, value FROM v_points WHERE metric = 'tokens' AND status = 'ok' ORDER BY ts",
		},
		{
			name:  "latest_per_context",
			query: "SELECT repo_id, rev_id, agent_id, question_id, dims, scope, arg_max(value, ts) FROM v_points WHERE metric = 'tokens' AND status = 'ok' GROUP BY repo_id, rev_id, agent_id, question_id, dims, scope",
		},
		{
			name:  "compare_runs",
			query: "SELECT run_id, value FROM v_points WHERE metric = 'tokens' AND repo_id = ? AND rev_id = ? ORDER BY run_id",
			args:  []interface{}{data.RepoID, data.FirstRevID},
		},
	}
	for _, q := range queries {
		duration, err := measureQuery(ctx, db, q.query, q.args...)
		if err != nil {
			t.Fatalf("query %s failed: %v", q.name, err)
		}
		t.Logf("%s: %s", q.name, duration)
		if duration > 5*time.Second {
			t.Fatalf("query %s exceeded 5s: %s", q.name, duration)
		}
	}
}
