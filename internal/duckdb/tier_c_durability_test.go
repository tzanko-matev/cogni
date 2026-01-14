//go:build duckdbtierc

package duckdb_test

import (
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/duckdb/testing"
	"cogni/internal/testutil"

	"github.com/google/uuid"
)

// TestTierCDurabilitySmoke verifies committed data persists on disk.
func TestTierCDurabilitySmoke(t *testing.T) {
	ctx := testutil.Context(t, 15*time.Second)
	dbPath := filepath.Join(t.TempDir(), "durable.duckdb")
	db := duckdbtesting.Open(t, dbPath)
	duckdbtesting.ApplySchema(t, db)

	repoID := uuid.NewString()
	if _, err := db.ExecContext(ctx, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, 'repo', 'git')", repoID); err != nil {
		t.Fatalf("insert repo: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	verifyDB := duckdbtesting.Open(t, dbPath)
	count := queryInt(t, ctx, verifyDB, "SELECT COUNT(*) FROM repos")
	if count != 1 {
		t.Fatalf("expected 1 repo after reopen, got %d", count)
	}

	tx, err := verifyDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, 'repo2', 'git')", uuid.NewString()); err != nil {
		t.Fatalf("insert repo2: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if err := verifyDB.Close(); err != nil {
		t.Fatalf("close verify db: %v", err)
	}

	finalDB := duckdbtesting.Open(t, dbPath)
	finalCount := queryInt(t, ctx, finalDB, "SELECT COUNT(*) FROM repos")
	if finalCount != 1 {
		t.Fatalf("expected 1 repo after rollback, got %d", finalCount)
	}
}
