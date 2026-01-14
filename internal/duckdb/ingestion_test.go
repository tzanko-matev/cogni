package duckdb_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"cogni/internal/duckdb"
	"cogni/internal/duckdb/testing"
	"cogni/internal/testutil"

	"github.com/google/uuid"
)

const (
	unitTimeout = 2 * time.Second
)

// TestCanonicalJSONStable verifies canonical JSON output ignores map key order.
func TestCanonicalJSONStable(t *testing.T) {
	ctx := testutil.Context(t, time.Second)
	runWithTimeout(t, ctx, func() error {
		specA := map[string]interface{}{
			"model": "gpt-test",
			"params": map[string]interface{}{
				"top_p": 1.0,
				"temp":  0.2,
			},
			"tags": []interface{}{"alpha", "beta"},
		}
		specB := map[string]interface{}{
			"tags": []interface{}{"alpha", "beta"},
			"params": map[string]interface{}{
				"temp":  0.2,
				"top_p": 1.0,
			},
			"model": "gpt-test",
		}
		left, err := duckdb.CanonicalJSON(specA)
		if err != nil {
			return fmt.Errorf("canonical json a: %w", err)
		}
		right, err := duckdb.CanonicalJSON(specB)
		if err != nil {
			return fmt.Errorf("canonical json b: %w", err)
		}
		if string(left) != string(right) {
			return fmt.Errorf("canonical json mismatch: %s vs %s", string(left), string(right))
		}
		return nil
	})
}

// TestUpsertHelpersIdempotent verifies upsert helpers deduplicate records by key.
func TestUpsertHelpersIdempotent(t *testing.T) {
	ctx := testutil.Context(t, unitTimeout)
	runWithTimeout(t, ctx, func() error {
		db := duckdbtesting.Open(t, ":memory:")
		duckdbtesting.ApplySchema(t, db)

		agentSpec := map[string]interface{}{"model": "agent-test"}
		agentID1, agentKey1, err := duckdb.UpsertAgent(ctx, db, agentSpec, "Agent")
		if err != nil {
			return fmt.Errorf("upsert agent: %w", err)
		}
		agentID2, agentKey2, err := duckdb.UpsertAgent(ctx, db, agentSpec, "Agent")
		if err != nil {
			return fmt.Errorf("upsert agent again: %w", err)
		}
		if agentKey1 != agentKey2 {
			return fmt.Errorf("agent keys mismatch: %s vs %s", agentKey1, agentKey2)
		}
		if agentID1 != agentID2 {
			return fmt.Errorf("agent ids mismatch: %s vs %s", agentID1, agentID2)
		}
		if err := assertRowCount(ctx, db, "agents", 1); err != nil {
			return err
		}

		questionSpec := map[string]interface{}{"title": "question-test"}
		questionID1, questionKey1, err := duckdb.UpsertQuestion(ctx, db, questionSpec, "Question")
		if err != nil {
			return fmt.Errorf("upsert question: %w", err)
		}
		questionID2, questionKey2, err := duckdb.UpsertQuestion(ctx, db, questionSpec, "Question")
		if err != nil {
			return fmt.Errorf("upsert question again: %w", err)
		}
		if questionKey1 != questionKey2 {
			return fmt.Errorf("question keys mismatch: %s vs %s", questionKey1, questionKey2)
		}
		if questionID1 != questionID2 {
			return fmt.Errorf("question ids mismatch: %s vs %s", questionID1, questionID2)
		}
		if err := assertRowCount(ctx, db, "questions", 1); err != nil {
			return err
		}

		repoID := uuid.NewString()
		revID := "rev-1"
		contextID1, contextKey1, err := duckdb.UpsertContext(ctx, db, duckdb.ContextInput{
			RepoID:      repoID,
			RevID:       revID,
			AgentID:     &agentID1,
			QuestionID:  &questionID1,
			AgentKey:    agentKey1,
			QuestionKey: questionKey1,
			Dims:        map[string]string{"benchmark": "tiny"},
			Scope:       map[string]interface{}{"kind": "path", "path": "src/main.go"},
		})
		if err != nil {
			return fmt.Errorf("upsert context: %w", err)
		}
		contextID2, contextKey2, err := duckdb.UpsertContext(ctx, db, duckdb.ContextInput{
			RepoID:      repoID,
			RevID:       revID,
			AgentID:     &agentID1,
			QuestionID:  &questionID1,
			AgentKey:    agentKey1,
			QuestionKey: questionKey1,
			Dims:        map[string]string{"benchmark": "tiny"},
			Scope:       map[string]interface{}{"path": "src/main.go", "kind": "path"},
		})
		if err != nil {
			return fmt.Errorf("upsert context again: %w", err)
		}
		if contextKey1 != contextKey2 {
			return fmt.Errorf("context keys mismatch: %s vs %s", contextKey1, contextKey2)
		}
		if contextID1 != contextID2 {
			return fmt.Errorf("context ids mismatch: %s vs %s", contextID1, contextID2)
		}
		if err := assertRowCount(ctx, db, "contexts", 1); err != nil {
			return err
		}
		return nil
	})
}

// runWithTimeout ensures a test body finishes before the context deadline.
func runWithTimeout(t *testing.T, ctx context.Context, fn func() error) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()
	select {
	case <-ctx.Done():
		t.Fatalf("test timed out: %v", ctx.Err())
	case err := <-done:
		if err != nil {
			t.Fatalf("test failed: %v", err)
		}
	}
}

// assertRowCount checks the expected row count for a table.
func assertRowCount(ctx context.Context, db *sql.DB, table string, want int) error {
	var got int
	query := "SELECT COUNT(*) FROM " + table
	if err := db.QueryRowContext(ctx, query).Scan(&got); err != nil {
		return fmt.Errorf("count %s: %w", table, err)
	}
	if got != want {
		return fmt.Errorf("%s row count: got %d want %d", table, got, want)
	}
	return nil
}
