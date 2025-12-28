package runner

import (
	"fmt"
	"path/filepath"
	"strings"
)

type OutputPaths struct {
	Root   string
	Commit string
	RunID  string
}

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

func (o OutputPaths) RunDir() string {
	return filepath.Join(o.Root, o.Commit, o.RunID)
}

func (o OutputPaths) ResultsPath() string {
	return filepath.Join(o.RunDir(), "results.json")
}

func (o OutputPaths) ReportPath() string {
	return filepath.Join(o.RunDir(), "report.html")
}

func (o OutputPaths) LogsDir() string {
	return filepath.Join(o.RunDir(), "logs")
}
