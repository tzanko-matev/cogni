//go:build duckdbtierc

package duckdb_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// fixtureConfig describes a Tier C fixture specification.
type fixtureConfig struct {
	Name      string `json:"name"`
	Revisions int    `json:"revisions"`
	Metrics   int    `json:"metrics"`
	Runs      int    `json:"runs"`
}

// fixtureData captures identifiers used by Tier C queries.
type fixtureData struct {
	RepoID         string
	RunIDs         []string
	MetricIDs      []string
	FirstContextID string
	FirstRevID     string
}

// fixtureNamespace ensures deterministic UUID generation.
var fixtureNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// loadFixtureConfig reads a fixture config from tests/fixtures/duckdb.
func loadFixtureConfig(name string) (fixtureConfig, error) {
	root, err := repoRoot()
	if err != nil {
		return fixtureConfig{}, err
	}
	path := filepath.Join(root, "tests", "fixtures", "duckdb", name+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fixtureConfig{}, err
	}
	var cfg fixtureConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fixtureConfig{}, err
	}
	if cfg.Name == "" {
		cfg.Name = name
	}
	return cfg, nil
}

// loadFixture inserts a deterministic dataset for Tier C tests.
func loadFixture(ctx context.Context, db *sql.DB, cfg fixtureConfig) (fixtureData, error) {
	repoID := deterministicID("repo", 0)
	if _, err := db.ExecContext(ctx, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, ?, 'git')", repoID, "fixture-"+cfg.Name); err != nil {
		return fixtureData{}, err
	}
	runIDs := make([]string, 0, cfg.Runs)
	for i := 0; i < cfg.Runs; i++ {
		runID := deterministicID("run", i)
		runIDs = append(runIDs, runID)
		if _, err := db.ExecContext(ctx, "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')", runID, repoID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			return fixtureData{}, err
		}
	}
	metricIDs := make([]string, 0, cfg.Metrics)
	for i := 0; i < cfg.Metrics; i++ {
		metricID := deterministicID("metric", i)
		name := fmt.Sprintf("metric_%d", i)
		if i == 0 {
			name = "tokens"
		}
		metricIDs = append(metricIDs, metricID)
		if _, err := db.ExecContext(ctx, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, ?, 'BIGINT')", metricID, name); err != nil {
			return fixtureData{}, err
		}
	}
	firstContextID := deterministicID("context", 0)
	firstRevID := fmt.Sprintf("rev-%06d", 0)
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fixtureData{}, err
	}
	defer tx.Rollback()
	revStmt, err := tx.PrepareContext(ctx, "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)")
	if err != nil {
		return fixtureData{}, err
	}
	defer revStmt.Close()
	ctxStmt, err := tx.PrepareContext(ctx, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fixtureData{}, err
	}
	defer ctxStmt.Close()
	measStmt, err := tx.PrepareContext(ctx, "INSERT INTO measurements (run_id, context_id, metric_id, value_bigint) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fixtureData{}, err
	}
	defer measStmt.Close()
	for i := 0; i < cfg.Revisions; i++ {
		revID := fmt.Sprintf("rev-%06d", i)
		ts := startTime.Add(time.Duration(i) * time.Minute)
		if _, err := revStmt.ExecContext(ctx, repoID, revID, ts); err != nil {
			return fixtureData{}, err
		}
		contextID := deterministicID("context", i)
		contextKey := deterministicID("context-key", i)
		if _, err := ctxStmt.ExecContext(ctx, contextID, contextKey, repoID, revID); err != nil {
			return fixtureData{}, err
		}
		for _, runID := range runIDs {
			for metricIndex, metricID := range metricIDs {
				value := int64(i + metricIndex)
				if _, err := measStmt.ExecContext(ctx, runID, contextID, metricID, value); err != nil {
					return fixtureData{}, err
				}
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fixtureData{}, err
	}
	return fixtureData{RepoID: repoID, RunIDs: runIDs, MetricIDs: metricIDs, FirstContextID: firstContextID, FirstRevID: firstRevID}, nil
}

// deterministicID generates a repeatable UUID from a prefix and index.
func deterministicID(prefix string, index int) string {
	return uuid.NewSHA1(fixtureNamespace, []byte(fmt.Sprintf("%s-%d", prefix, index))).String()
}
