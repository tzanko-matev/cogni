package runner

import (
	"path/filepath"
	"testing"
)

func TestOutputPaths(t *testing.T) {
	root := t.TempDir()
	paths, err := NewOutputPaths(root, "abc123", "20240102T030405Z-deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedRunDir := filepath.Join(root, "abc123", "20240102T030405Z-deadbeef")
	if paths.RunDir() != expectedRunDir {
		t.Fatalf("unexpected run dir: %q", paths.RunDir())
	}
	if paths.ResultsPath() != filepath.Join(expectedRunDir, "results.json") {
		t.Fatalf("unexpected results path: %q", paths.ResultsPath())
	}
	if paths.ReportPath() != filepath.Join(expectedRunDir, "report.html") {
		t.Fatalf("unexpected report path: %q", paths.ReportPath())
	}
	if paths.LogsDir() != filepath.Join(expectedRunDir, "logs") {
		t.Fatalf("unexpected logs dir: %q", paths.LogsDir())
	}
}

func TestOutputPathsErrors(t *testing.T) {
	cases := []struct {
		name   string
		root   string
		commit string
		runID  string
	}{
		{name: "missing-root", root: "", commit: "abc", runID: "id"},
		{name: "missing-commit", root: "out", commit: "", runID: "id"},
		{name: "missing-run", root: "out", commit: "abc", runID: ""},
	}
	for _, tc := range cases {
		if _, err := NewOutputPaths(tc.root, tc.commit, tc.runID); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}
