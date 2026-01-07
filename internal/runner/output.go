package runner

import (
	"fmt"
	"path/filepath"
	"strings"
)

// OutputPaths describes filesystem locations for run outputs.
type OutputPaths struct {
	Root   string
	Commit string
	RunID  string
}

// NewOutputPaths validates and constructs output paths metadata.
func NewOutputPaths(root, commit, runID string) (OutputPaths, error) {
	if strings.TrimSpace(root) == "" {
		return OutputPaths{}, fmt.Errorf("output root is empty")
	}
	if strings.TrimSpace(commit) == "" {
		return OutputPaths{}, fmt.Errorf("commit is empty")
	}
	if strings.TrimSpace(runID) == "" {
		return OutputPaths{}, fmt.Errorf("run ID is empty")
	}
	return OutputPaths{
		Root:   root,
		Commit: commit,
		RunID:  runID,
	}, nil
}

// RunDir returns the directory for a specific run.
func (o OutputPaths) RunDir() string {
	return filepath.Join(o.Root, o.Commit, o.RunID)
}

// ResultsPath returns the path to results.json.
func (o OutputPaths) ResultsPath() string {
	return filepath.Join(o.RunDir(), "results.json")
}

// ReportPath returns the path to the HTML report.
func (o OutputPaths) ReportPath() string {
	return filepath.Join(o.RunDir(), "report.html")
}

// LogsDir returns the path for log outputs.
func (o OutputPaths) LogsDir() string {
	return filepath.Join(o.RunDir(), "logs")
}
