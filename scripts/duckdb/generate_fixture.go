package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cogni/internal/duckdb"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
)

// fixtureConfig defines the JSON config for generating a DuckDB fixture.
type fixtureConfig struct {
	Name      string `json:"name"`
	Revisions int    `json:"revisions"`
	Metrics   int    `json:"metrics"`
	Runs      int    `json:"runs"`
}

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

func generateFixture(ctx context.Context, path string, cfg fixtureConfig) error {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := duckdb.EnsureSchema(db); err != nil {
		return err
	}
	repoID := deterministicID("repo", 0)
	if _, err := db.ExecContext(ctx, "INSERT INTO repos (repo_id, name, vcs) VALUES (?, ?, 'git')", repoID, "fixture-"+cfg.Name); err != nil {
		return err
	}
	runIDs := make([]string, 0, cfg.Runs)
	for i := 0; i < cfg.Runs; i++ {
		runID := deterministicID("run", i)
		runIDs = append(runIDs, runID)
		if _, err := db.ExecContext(ctx, "INSERT INTO runs (run_id, repo_id, collected_at, tool_name) VALUES (?, ?, ?, 'cogni')", runID, repoID, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)); err != nil {
			return err
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
			return err
		}
	}
	startTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	revStmt, err := tx.PrepareContext(ctx, "INSERT INTO revisions (repo_id, rev_id, ts_utc) VALUES (?, ?, ?)")
	if err != nil {
		return err
	}
	defer revStmt.Close()
	ctxStmt, err := tx.PrepareContext(ctx, "INSERT INTO contexts (context_id, context_key, repo_id, rev_id) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer ctxStmt.Close()
	measStmt, err := tx.PrepareContext(ctx, "INSERT INTO measurements (run_id, context_id, metric_id, value_bigint) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer measStmt.Close()
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
		for _, runID := range runIDs {
			for metricIndex, metricID := range metricIDs {
				value := int64(i + metricIndex)
				if _, err := measStmt.ExecContext(ctx, runID, contextID, metricID, value); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

func deterministicID(prefix string, index int) string {
	return uuid.NewSHA1(fixtureNamespace, []byte(fmt.Sprintf("%s-%d", prefix, index))).String()
}

func dirOf(path string) string {
	if path == "" {
		return "."
	}
	if idx := len(path) - 1; idx >= 0 && path[idx] == os.PathSeparator {
		return path
	}
	return filepath.Dir(path)
}

var fixtureNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
