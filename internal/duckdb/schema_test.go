package duckdb_test

import "testing"

// TestSchemaObjectsExist verifies core tables and views are created.
func TestSchemaObjectsExist(t *testing.T) {
	db, ctx := openTestDB(t)
	for _, table := range []string{
		"repos",
		"revisions",
		"revision_parents",
		"metric_defs",
		"runs",
		"agents",
		"questions",
		"contexts",
		"measurements",
		"derived_metric_defs",
	} {
		count := queryInt(t, ctx, db, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = ?", table)
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}
	viewCount := queryInt(t, ctx, db, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'v_points' AND table_type = 'VIEW'")
	if viewCount != 1 {
		t.Fatalf("expected view v_points to exist")
	}
	execSQL(t, ctx, db, "SELECT * FROM v_points LIMIT 0")
}
