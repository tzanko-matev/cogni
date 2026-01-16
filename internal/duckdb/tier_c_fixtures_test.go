//go:build duckdbtierc

package duckdb_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	duckdb "github.com/duckdb/duckdb-go/v2"
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
	conn, err := db.Conn(ctx)
	if err != nil {
		return fixtureData{}, err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return fixtureData{}, err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
	}()
	repoID := deterministicID("repo", 0)
	if _, err := conn.ExecContext(ctx, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, ?, 'git')", repoID, "fixture-"+cfg.Name); err != nil {
		return fixtureData{}, err
	}
	runIDs := make([]string, 0, cfg.Runs)
	runUUIDs := make([]duckdb.UUID, 0, cfg.Runs)
	for i := 0; i < cfg.Runs; i++ {
		runID := deterministicID("run", i)
		runUUID, err := parseDuckDBUUID(runID)
		if err != nil {
			return fixtureData{}, err
		}
		runIDs = append(runIDs, runID)
		runUUIDs = append(runUUIDs, runUUID)
		if _, err := conn.ExecContext(ctx, "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')", runID, repoID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			return fixtureData{}, err
		}
	}
	metricIDs := make([]string, 0, cfg.Metrics)
	metricUUIDs := make([]duckdb.UUID, 0, cfg.Metrics)
	for i := 0; i < cfg.Metrics; i++ {
		metricID := deterministicID("metric", i)
		metricUUID, err := parseDuckDBUUID(metricID)
		if err != nil {
			return fixtureData{}, err
		}
		name := fmt.Sprintf("metric_%d", i)
		if i == 0 {
			name = "tokens"
		}
		metricIDs = append(metricIDs, metricID)
		metricUUIDs = append(metricUUIDs, metricUUID)
		if _, err := conn.ExecContext(ctx, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, ?, 'BIGINT')", metricID, name); err != nil {
			return fixtureData{}, err
		}
	}
	firstContextID := deterministicID("context", 0)
	firstRevID := fmt.Sprintf("rev-%06d", 0)
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	revStmt, err := conn.PrepareContext(ctx, "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)")
	if err != nil {
		return fixtureData{}, err
	}
	defer revStmt.Close()
	ctxStmt, err := conn.PrepareContext(ctx, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fixtureData{}, err
	}
	defer ctxStmt.Close()
	measurementAppender, err := newMeasurementAppender(conn)
	if err != nil {
		return fixtureData{}, err
	}
	defer func() {
		if measurementAppender != nil {
			_ = measurementAppender.Close()
		}
	}()
	const sampleIndex = int32(0)
	const statusOK = "ok"
	var nullValue driver.Value
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
		contextUUID, err := parseDuckDBUUID(contextID)
		if err != nil {
			return fixtureData{}, err
		}
		for _, runUUID := range runUUIDs {
			for metricIndex, metricUUID := range metricUUIDs {
				value := int64(i + metricIndex)
				if err := measurementAppender.AppendRow(
					runUUID,
					contextUUID,
					metricUUID,
					sampleIndex,
					nullValue,
					nullValue,
					value,
					nullValue,
					nullValue,
					nullValue,
					nullValue,
					statusOK,
					nullValue,
					nullValue,
				); err != nil {
					return fixtureData{}, err
				}
			}
		}
	}
	if err := measurementAppender.Close(); err != nil {
		return fixtureData{}, err
	}
	measurementAppender = nil
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fixtureData{}, err
	}
	committed = true
	return fixtureData{RepoID: repoID, RunIDs: runIDs, MetricIDs: metricIDs, FirstContextID: firstContextID, FirstRevID: firstRevID}, nil
}
