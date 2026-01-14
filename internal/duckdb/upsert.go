package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// ContextKeyInput describes the fields used to compute a stable context key.
type ContextKeyInput struct {
	RepoID      string
	RevID       string
	AgentKey    string
	QuestionKey string
	Dims        map[string]string
	Scope       interface{}
}

// ContextInput describes the record to insert or upsert into contexts.
type ContextInput struct {
	RepoID      string
	RevID       string
	AgentID     *string
	QuestionID  *string
	AgentKey    string
	QuestionKey string
	Dims        map[string]string
	Scope       interface{}
}

// ContextKey returns a deterministic fingerprint for a context record.
func ContextKey(input ContextKeyInput) (string, error) {
	payload := map[string]interface{}{
		"repo_id":      input.RepoID,
		"rev_id":       input.RevID,
		"agent_key":    input.AgentKey,
		"question_key": input.QuestionKey,
		"dims":         CanonicalDims(input.Dims),
		"scope":        input.Scope,
	}
	return FingerprintJSON(payload)
}

// UpsertAgent inserts or updates an agent by its fingerprint key.
func UpsertAgent(ctx context.Context, db *sql.DB, spec interface{}, displayName string) (string, string, error) {
	if ctx == nil {
		return "", "", errors.New("duckdb: context is nil")
	}
	if db == nil {
		return "", "", errors.New("duckdb: db is nil")
	}
	if spec == nil {
		return "", "", errors.New("duckdb: agent spec is nil")
	}
	canonical, err := CanonicalJSON(spec)
	if err != nil {
		return "", "", err
	}
	key := fingerprintBytes(canonical)
	id := uuid.NewString()
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO agents (agent_id, agent_key, spec, display_name, created_at)
		 VALUES (?, ?, ?, ?, now())
		 ON CONFLICT (agent_key) DO NOTHING`,
		id,
		key,
		string(canonical),
		displayName,
	); err != nil {
		return "", "", fmt.Errorf("upsert agent: %w", err)
	}
	outID, err := lookupID(ctx, db, "agents", "agent_id", "agent_key", key)
	if err != nil {
		return "", "", fmt.Errorf("lookup agent id: %w", err)
	}
	return outID, key, nil
}

// UpsertQuestion inserts or updates a question by its fingerprint key.
func UpsertQuestion(ctx context.Context, db *sql.DB, spec interface{}, title string) (string, string, error) {
	if ctx == nil {
		return "", "", errors.New("duckdb: context is nil")
	}
	if db == nil {
		return "", "", errors.New("duckdb: db is nil")
	}
	if spec == nil {
		return "", "", errors.New("duckdb: question spec is nil")
	}
	canonical, err := CanonicalJSON(spec)
	if err != nil {
		return "", "", err
	}
	key := fingerprintBytes(canonical)
	id := uuid.NewString()
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO questions (question_id, question_key, spec, title, created_at)
		 VALUES (?, ?, ?, ?, now())
		 ON CONFLICT (question_key) DO NOTHING`,
		id,
		key,
		string(canonical),
		title,
	); err != nil {
		return "", "", fmt.Errorf("upsert question: %w", err)
	}
	outID, err := lookupID(ctx, db, "questions", "question_id", "question_key", key)
	if err != nil {
		return "", "", fmt.Errorf("lookup question id: %w", err)
	}
	return outID, key, nil
}

// UpsertContext inserts or updates a context by its fingerprint key.
func UpsertContext(ctx context.Context, db *sql.DB, input ContextInput) (string, string, error) {
	if ctx == nil {
		return "", "", errors.New("duckdb: context is nil")
	}
	if db == nil {
		return "", "", errors.New("duckdb: db is nil")
	}
	if input.RepoID == "" || input.RevID == "" {
		return "", "", errors.New("duckdb: repo_id and rev_id are required")
	}
	key, err := ContextKey(ContextKeyInput{
		RepoID:      input.RepoID,
		RevID:       input.RevID,
		AgentKey:    input.AgentKey,
		QuestionKey: input.QuestionKey,
		Dims:        input.Dims,
		Scope:       input.Scope,
	})
	if err != nil {
		return "", "", err
	}
	var scopeValue interface{}
	if input.Scope != nil {
		canonical, err := CanonicalJSON(input.Scope)
		if err != nil {
			return "", "", err
		}
		scopeValue = string(canonical)
	}
	mapExpr := mapExpression(input.Dims)
	query := fmt.Sprintf(
		`INSERT INTO contexts (
		  context_id, context_key, repo_id, rev_id, agent_id, question_id, dims, scope, created_at
		) VALUES (?, ?, ?, ?, ?, ?, %s, ?, now())
		ON CONFLICT (context_key) DO NOTHING`,
		mapExpr,
	)
	id := uuid.NewString()
	args := []interface{}{id, key, input.RepoID, input.RevID, nullableString(input.AgentID), nullableString(input.QuestionID)}
	args = append(args, scopeValue)
	if _, err := db.ExecContext(ctx, query, args...); err != nil {
		return "", "", fmt.Errorf("upsert context: %w", err)
	}
	outID, err := lookupID(ctx, db, "contexts", "context_id", "context_key", key)
	if err != nil {
		return "", "", fmt.Errorf("lookup context id: %w", err)
	}
	return outID, key, nil
}
