package report

import (
	"strings"
	"testing"

	"cogni/internal/runner"
)

// TestResolveRunByCommitAndRunID verifies run resolution by commit and run ID.
func TestResolveRunByCommitAndRunID(t *testing.T) {
	root := t.TempDir()
	first := runner.Results{
		RunID: "run-1",
		Repo:  runner.RepoMetadata{Commit: "abc"},
	}
	if _, err := runner.WriteRunOutputs(first, root); err != nil {
		t.Fatalf("write outputs: %v", err)
	}
	second := runner.Results{
		RunID: "run-2",
		Repo:  runner.RepoMetadata{Commit: "def"},
	}
	if _, err := runner.WriteRunOutputs(second, root); err != nil {
		t.Fatalf("write outputs: %v", err)
	}

	resolved, _, err := ResolveRun(root, "", "abc")
	if err != nil {
		t.Fatalf("resolve commit: %v", err)
	}
	if resolved.RunID != "run-1" {
		t.Fatalf("unexpected run id: %s", resolved.RunID)
	}

	resolved, _, err = ResolveRun(root, "", "run-2")
	if err != nil {
		t.Fatalf("resolve run id: %v", err)
	}
	if resolved.Repo.Commit != "def" {
		t.Fatalf("unexpected commit: %s", resolved.Repo.Commit)
	}
}

// TestBuildReportHTML verifies report HTML includes run metadata.
func TestBuildReportHTML(t *testing.T) {
	runs := []runner.Results{
		{RunID: "run-1", Repo: runner.RepoMetadata{Commit: "abc"}},
		{RunID: "run-2", Repo: runner.RepoMetadata{Commit: "def"}},
	}
	html := BuildReportHTML(runs)
	for _, token := range []string{"abc", "def", "run-1", "run-2"} {
		if !strings.Contains(html, token) {
			t.Fatalf("expected report to include %s", token)
		}
	}
	if !strings.Contains(html, "<table") {
		t.Fatalf("expected table in report")
	}
}
