package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"cogni/internal/duckdb"
)

// fixtureConfig defines the JSON config for generating a DuckDB fixture.
type fixtureConfig struct {
	Name      string `json:"name"`
	Revisions int    `json:"revisions"`
	Metrics   int    `json:"metrics"`
	Runs      int    `json:"runs"`
}

// main generates an on-disk DuckDB fixture from a JSON config.
func main() {
	configPath := flag.String("config", "", "path to fixture config JSON")
	outPath := flag.String("out", "", "output duckdb file path")
	flag.Parse()
	if *configPath == "" || *outPath == "" {
		fmt.Fprintln(os.Stderr, "usage: generate_fixture --config <path> --out <duckdb file>")
		os.Exit(2)
	}
	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(dirOf(*outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output dir: %v\n", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := generateFixture(ctx, *outPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "generate fixture: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig parses a fixture JSON config from disk.
func loadConfig(path string) (fixtureConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return fixtureConfig{}, err
	}
	var cfg fixtureConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fixtureConfig{}, err
	}
	return cfg, nil
}

// generateFixture creates and populates the DuckDB file at path.
func generateFixture(ctx context.Context, path string, cfg fixtureConfig) error {
	if err := removeIfExists(path); err != nil {
		return err
	}
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := duckdb.EnsureSchema(db); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, "BEGIN"); err != nil {
		return err
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
		return err
	}
	runIDs := make([]string, 0, cfg.Runs)
	runUUIDs := make([]duckdbUUID, 0, cfg.Runs)
	for i := 0; i < cfg.Runs; i++ {
		runID := deterministicID("run", i)
		runUUID, err := parseDuckDBUUID(runID)
		if err != nil {
			return err
		}
		runIDs = append(runIDs, runID)
		runUUIDs = append(runUUIDs, runUUID)
		if _, err := conn.ExecContext(ctx, "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')", runID, repoID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			return err
		}
	}
	metricIDs := make([]string, 0, cfg.Metrics)
	metricUUIDs := make([]duckdbUUID, 0, cfg.Metrics)
	for i := 0; i < cfg.Metrics; i++ {
		metricID := deterministicID("metric", i)
		metricUUID, err := parseDuckDBUUID(metricID)
		if err != nil {
			return err
		}
		name := fmt.Sprintf("metric_%d", i)
		if i == 0 {
			name = "tokens"
		}
		metricIDs = append(metricIDs, metricID)
		metricUUIDs = append(metricUUIDs, metricUUID)
		if _, err := conn.ExecContext(ctx, "INSERT INTO metric_defs (metric_id, name, physical_type) VALUES (?, ?, 'BIGINT')", metricID, name); err != nil {
			return err
		}
	}
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	revStmt, err := conn.PrepareContext(ctx, "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer revStmt.Close()
	ctxStmt, err := conn.PrepareContext(ctx, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer ctxStmt.Close()
	measurementAppender, err := newMeasurementAppender(conn)
	if err != nil {
		return err
	}
	defer func() {
		if measurementAppender != nil {
			_ = measurementAppender.Close()
		}
	}()
	const sampleIndex = int32(0)
	const statusOK = "ok"
	var nullValue any
	for i := 0; i < cfg.Revisions; i++ {
		revID := fmt.Sprintf("rev-%06d", i)
		ts := startTime.Add(time.Duration(i) * time.Minute)
		if _, err := revStmt.ExecContext(ctx, repoID, revID, ts); err != nil {
			return err
		}
		contextID := deterministicID("context", i)
		contextKey := deterministicID("context-key", i)
		if _, err := ctxStmt.ExecContext(ctx, contextID, contextKey, repoID, revID); err != nil {
			return err
		}
		contextUUID, err := parseDuckDBUUID(contextID)
		if err != nil {
			return err
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
					return err
				}
			}
		}
	}
	if err := measurementAppender.Close(); err != nil {
		return err
	}
	measurementAppender = nil
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}
