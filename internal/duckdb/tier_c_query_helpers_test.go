//go:build duckdbtierc

package duckdb_test

import (
	"context"
	"database/sql"
	"time"
)

// measureQuery runs a query, drains all rows, and returns elapsed time.
func measureQuery(ctx context.Context, db *sql.DB, query string, args ...interface{}) (time.Duration, error) {
	start := time.Now()
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if err := drainRows(rows); err != nil {
		return 0, err
	}
	return time.Since(start), nil
}

// drainRows consumes all rows to include scan time in benchmarks.
func drainRows(rows *sql.Rows) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}
	dest := make([]interface{}, len(cols))
	for i := range dest {
		var holder interface{}
		dest[i] = &holder
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
	}
	return rows.Err()
}
