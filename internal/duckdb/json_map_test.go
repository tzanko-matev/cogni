package duckdb_test

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
)

// TestJSONAndMapSemantics validates JSON parsing and MAP access behavior.
func TestJSONAndMapSemantics(t *testing.T) {
	db, ctx := openTestDB(t)
	agentID := uuid.NewString()
	execSQL(t, ctx, db, "INSERT INTO agents (agent_id, agent_key, spec) VALUES (?, 'agent-json', '{\"model\":\"gpt-4\"}')", agentID)

	var model sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT spec->>'$.model' FROM agents WHERE agent_key = 'agent-json'").Scan(&model); err != nil {
		t.Fatalf("json extract failed: %v", err)
	}
	if !model.Valid || model.String != "gpt-4" {
		t.Fatalf("expected model gpt-4, got %v", model.String)
	}

	if _, err := db.ExecContext(ctx, "INSERT INTO agents (agent_id, agent_key, spec) VALUES (?, 'agent-bad-json', '{invalid}')", uuid.NewString()); err == nil {
		t.Fatalf("expected invalid JSON insert to fail")
	}

	execSQL(t, ctx, db, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id, dims) VALUES (?, 'context-null', ?, 'rev', NULL)", uuid.NewString(), uuid.NewString())
	execSQL(t, ctx, db, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id, dims) VALUES (?, 'context-empty', ?, 'rev', map([], []))", uuid.NewString(), uuid.NewString())
	execSQL(t, ctx, db, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id, dims) VALUES (?, 'context-dims', ?, 'rev', map(['benchmark'], ['tiny']))", uuid.NewString(), uuid.NewString())

	var benchmark sql.NullString
	if err := db.QueryRowContext(ctx, "SELECT dims['benchmark'] FROM contexts WHERE context_key = 'context-dims'").Scan(&benchmark); err != nil {
		t.Fatalf("map extract failed: %v", err)
	}
	if !benchmark.Valid || benchmark.String != "tiny" {
		t.Fatalf("expected benchmark tiny, got %v", benchmark.String)
	}

	if err := db.QueryRowContext(ctx, "SELECT dims['missing'] FROM contexts WHERE context_key = 'context-dims'").Scan(&benchmark); err != nil {
		t.Fatalf("map missing key failed: %v", err)
	}
	if benchmark.Valid {
		t.Fatalf("expected missing key to return NULL")
	}

	if err := db.QueryRowContext(ctx, "SELECT dims['benchmark'] FROM contexts WHERE context_key = 'context-null'").Scan(&benchmark); err != nil {
		t.Fatalf("map null dims failed: %v", err)
	}
	if benchmark.Valid {
		t.Fatalf("expected NULL dims to return NULL")
	}
}
