package duckdbtesting

import (
	"database/sql"
	"testing"
	"time"

	"cogni/internal/duckdb"
	"cogni/internal/testutil"

	_ "github.com/marcboeker/go-duckdb"
)

const (
	defaultTimeout = 2 * time.Second
)

// Open opens a DuckDB connection and verifies it responds within a short timeout.
func Open(t testing.TB, dsn string) *sql.DB {
	t.Helper()
	ctx := testutil.Context(t, defaultTimeout)
	conn, err := sql.Open("duckdb", dsn)
	if err != nil {
		t.Fatalf("open duckdb: %v", err)
	}
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		t.Fatalf("ping duckdb: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

// ApplySchema executes the DuckDB schema DDL on the provided connection.
func ApplySchema(t testing.TB, db *sql.DB) {
	t.Helper()
	ctx := testutil.Context(t, defaultTimeout)
	if _, err := db.ExecContext(ctx, duckdb.SchemaDDL()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
}
