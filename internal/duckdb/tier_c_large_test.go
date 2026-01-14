//go:build duckdbtierc && duckdbtierclarge

package duckdb_test

import (
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/duckdb/testing"
	"cogni/internal/testutil"
)

// TestTierCLargePerformance loads the large fixture for optional stress testing.
func TestTierCLargePerformance(t *testing.T) {
	ctx := testutil.Context(t, 120*time.Second)
	dbPath := filepath.Join(t.TempDir(), "large.duckdb")
	db := duckdbtesting.Open(t, dbPath)
	duckdbtesting.ApplySchema(t, db)

	cfg, err := loadFixtureConfig("large")
	if err != nil {
		t.Fatalf("load fixture config: %v", err)
	}
	data, err := loadFixture(ctx, db, cfg)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	duration, err := measureQuery(ctx, db, "SELECT ts, value FROM v_points WHERE metric = 'tokens' AND status = 'ok' ORDER BY ts")
	if err != nil {
		t.Fatalf("tokens query failed: %v", err)
	}
	t.Logf("tokens_over_time: %s", duration)

	duration, err = measureQuery(ctx, db, "SELECT run_id, value FROM v_points WHERE metric = 'tokens' AND repo_id = ? AND rev_id = ? ORDER BY run_id", data.RepoID, data.FirstRevID)
	if err != nil {
		t.Fatalf("compare runs failed: %v", err)
	}
	t.Logf("compare_runs: %s", duration)
}
