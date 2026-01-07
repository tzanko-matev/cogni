package runner

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteRunOutputs verifies output files and directories are created.
func TestWriteRunOutputs(t *testing.T) {
	root := t.TempDir()
	results := Results{
		RunID: "run-1",
		Repo: RepoMetadata{
			Commit: "abc123",
		},
	}
	paths, err := WriteRunOutputs(results, root)
	if err != nil {
		t.Fatalf("write outputs: %v", err)
	}
	if _, err := os.Stat(paths.ResultsPath()); err != nil {
		t.Fatalf("missing results.json: %v", err)
	}
	if _, err := os.Stat(paths.ReportPath()); err != nil {
		t.Fatalf("missing report.html: %v", err)
	}
	if _, err := os.Stat(paths.LogsDir()); err != nil {
		t.Fatalf("missing logs dir: %v", err)
	}
	expectedDir := filepath.Join(root, "abc123", "run-1")
	if paths.RunDir() != expectedDir {
		t.Fatalf("unexpected run dir: %s", paths.RunDir())
	}
}
