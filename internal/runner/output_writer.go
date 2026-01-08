package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WriteRunOutputs writes run outputs and prepares output directories.
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

// writeJSON writes a Results payload as pretty JSON.
func writeJSON(path string, results Results) error {
	payload, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	return nil
}

// writePlaceholderReport writes a minimal HTML report stub.
func writePlaceholderReport(path string, results Results) error {
	content, err := renderRunReportHTML(context.Background(), results)
	if err != nil {
		return fmt.Errorf("render report: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}
	return nil
}
