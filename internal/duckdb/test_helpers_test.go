package duckdb_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"cogni/internal/duckdb/testing"
	"cogni/internal/testutil"
)

const (
	testTimeout = 2 * time.Second
)

// openTestDB opens an in-memory DuckDB instance with the schema applied.
func openTestDB(t *testing.T) (*sql.DB, context.Context) {
	t.Helper()
	ctx := testutil.Context(t, testTimeout)
	db := duckdbtesting.Open(t, ":memory:")
	duckdbtesting.ApplySchema(t, db)
	return db, ctx
}

// execSQL executes a statement and fails the test on error.
func execSQL(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...interface{}) {
	t.Helper()
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		t.Fatalf("exec sql failed: %v", err)
	}
}

// queryInt returns a single integer value from the database.
func queryInt(t *testing.T, ctx context.Context, db *sql.DB, query string, args ...interface{}) int {
	t.Helper()
	var out int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&out); err != nil {
		t.Fatalf("query int failed: %v", err)
	}
	return out
}
