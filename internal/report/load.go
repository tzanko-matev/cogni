package report

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cogni/internal/runner"
	"cogni/internal/vcs"
)

func LoadResults(path string) (runner.Results, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return runner.Results{}, err
	}
	var results runner.Results
	if err := json.Unmarshal(data, &results); err != nil {
		return runner.Results{}, err
	}
	return results, nil
}

func ResolveRun(outputDir, repoRoot, ref string) (runner.Results, string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return runner.Results{}, "", fmt.Errorf("run ref is required")
	}
	commit := ref
	if repoRoot != "" {
		if resolved, err := vcs.ResolveRef(context.Background(), repoRoot, ref); err == nil {
			commit = resolved
		}
	}
	commitDir := filepath.Join(outputDir, commit)
	if info, err := os.Stat(commitDir); err == nil && info.IsDir() {
		runDir, err := findLatestRunDir(commitDir)
		if err != nil {
			return runner.Results{}, "", err
		}
		results, err := LoadResults(filepath.Join(runDir, "results.json"))
		return results, runDir, err
	}
	runDir, err := findRunByID(outputDir, ref)
	if err != nil {
		return runner.Results{}, "", err
	}
	results, err := LoadResults(filepath.Join(runDir, "results.json"))
	return results, runDir, err
}

func findLatestRunDir(commitDir string) (string, error) {
	entries, err := os.ReadDir(commitDir)
	if err != nil {
		return "", err
	}
	runIDs := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			runIDs = append(runIDs, entry.Name())
		}
	}
	if len(runIDs) == 0 {
		return "", fmt.Errorf("no runs found in %s", commitDir)
	}
	sort.Strings(runIDs)
	return filepath.Join(commitDir, runIDs[len(runIDs)-1]), nil
}

func findRunByID(outputDir, runID string) (string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runDir := filepath.Join(outputDir, entry.Name(), runID)
		if info, err := os.Stat(runDir); err == nil && info.IsDir() {
			return runDir, nil
		}
	}
	return "", fmt.Errorf("run %s not found", runID)
}
