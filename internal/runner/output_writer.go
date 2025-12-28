package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func WriteRunOutputs(results Results, outputDir string) (OutputPaths, error) {
	if outputDir == "" {
		return OutputPaths{}, fmt.Errorf("output directory is required")
	}
	paths, err := NewOutputPaths(outputDir, results.Repo.Commit, results.RunID)
	if err != nil {
		return OutputPaths{}, err
	}
	if err := os.MkdirAll(paths.RunDir(), 0o755); err != nil {
		return OutputPaths{}, fmt.Errorf("create output dir: %w", err)
	}
	if err := writeJSON(paths.ResultsPath(), results); err != nil {
		return OutputPaths{}, err
	}
	if err := writePlaceholderReport(paths.ReportPath(), results); err != nil {
		return OutputPaths{}, err
	}
	if err := os.MkdirAll(paths.LogsDir(), 0o755); err != nil {
		return OutputPaths{}, fmt.Errorf("create logs dir: %w", err)
	}
	return paths, nil
}

func writeJSON(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

func writePlaceholderReport(path string, results Results) error {
	content := fmt.Sprintf("<!doctype html>\n<html><head><meta charset=\"utf-8\"><title>Cogni Report</title></head><body><h1>Cogni Report</h1><p>Run %s for commit %s</p></body></html>\n", results.RunID, results.Repo.Commit)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
