package duckdb_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

type constraintCase struct {
	name  string
	query string
	args  []interface{}
}

// TestPrimaryKeyConstraints ensures primary keys reject duplicates.
func TestPrimaryKeyConstraints(t *testing.T) {
	db, ctx := openTestDB(t)
	now := time.Date(2026, 1, 14, 0, 0, 0, 0, time.UTC)
	cases := []constraintCase{
		{
			name:  "repos",
			query: "INSERT INTO repos (repo_id, name, vcs) VALUES (?, 'repo', 'git')",
			args:  []interface{}{uuid.NewString()},
		},
		{
			name:  "revisions",
			query: "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)",
			args:  []interface{}{uuid.NewString(), "rev-1", now},
		},
		{
			name:  "revision_parents",
			query: "INSERT INTO revision_parents (repo_id, child_rev_id, parent_rev_id) VALUES (?, ?, ?)",
			args:  []interface{}{uuid.NewString(), "child", "parent"},
		},
		{
			name:  "metric_defs",
			query: "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, ?, 'BIGINT')",
			args:  []interface{}{uuid.NewString(), "tokens"},
		},
		{
			name:  "runs",
			query: "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')",
			args:  []interface{}{uuid.NewString(), uuid.NewString(), now},
		},
		{
			name:  "agents",
			query: "INSERT INTO agents (agent_id, agent_key, spec) VALUES (?, ?, '{\"model\":\"gpt\"}')",
			args:  []interface{}{uuid.NewString(), "agent-key"},
		},
		{
			name:  "questions",
			query: "INSERT INTO questions (question_id, question_key, spec) VALUES (?, ?, '{\"title\":\"q\"}')",
			args:  []interface{}{uuid.NewString(), "question-key"},
		},
		{
			name:  "contexts",
			query: "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)",
			args:  []interface{}{uuid.NewString(), "context-key", uuid.NewString(), "rev"},
		},
		{
			name:  "measurements",
			query: "INSERT INTO measurements (run_id, context_id, metric_id, sample_index) VALUES (?, ?, ?, 0)",
			args:  []interface{}{uuid.NewString(), uuid.NewString(), uuid.NewString()},
		},
		{
			name:  "derived_metric_defs",
			query: "INSERT INTO derived_metric_defs (derived_metric_id, name, sql_select) VALUES (?, ?, 'SELECT 1')",
			args:  []interface{}{uuid.NewString(), "derived"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			execSQL(t, ctx, db, tc.query, tc.args...)
			if _, err := db.ExecContext(ctx, tc.query, tc.args...); err == nil {
				t.Fatalf("expected duplicate insert to fail for %s", tc.name)
			}
		})
	}
}

// TestUniqueConstraints ensures unique indexes reject duplicates.
func TestUniqueConstraints(t *testing.T) {
	db, ctx := openTestDB(t)
	cases := []constraintCase{
		{
			name:  "agents.agent_key",
			query: "INSERT INTO agents (agent_id, agent_key, spec) VALUES (?, ?, '{\"model\":\"gpt\"}')",
			args:  []interface{}{uuid.NewString(), "agent-key"},
		},
		{
			name:  "questions.question_key",
			query: "INSERT INTO questions (question_id, question_key, spec) VALUES (?, ?, '{\"title\":\"q\"}')",
			args:  []interface{}{uuid.NewString(), "question-key"},
		},
		{
			name:  "contexts.context_key",
			query: "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)",
			args:  []interface{}{uuid.NewString(), "context-key", uuid.NewString(), "rev"},
		},
		{
			name:  "metric_defs.name",
			query: "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, ?, 'BIGINT')",
			args:  []interface{}{uuid.NewString(), "tokens"},
		},
		{
			name:  "derived_metric_defs.name",
			query: "INSERT INTO derived_metric_defs (derived_metric_id, name, sql_select) VALUES (?, ?, 'SELECT 1')",
			args:  []interface{}{uuid.NewString(), "derived"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			execSQL(t, ctx, db, tc.query, tc.args...)
			secondArgs := append([]interface{}{}, tc.args...)
			secondArgs[0] = uuid.NewString()
			if _, err := db.ExecContext(ctx, tc.query, secondArgs...); err == nil {
				t.Fatalf("expected unique constraint to fail for %s", tc.name)
			}
		})
	}
}
