package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"cogni/internal/runner"
	"cogni/internal/vcs"
)

func TestCompareCommand(t *testing.T) {
	origResolve := resolveRun
	resolveRun = func(_ string, _ string, ref string) (runner.Results, string, error) {
		if ref == "base" {
			return runner.Results{RunID: "run-base", Repo: runner.RepoMetadata{Commit: "base"}, Summary: runner.RunSummary{PassRate: 0.5, TokensTotal: 10}}, "", nil
		}
		return runner.Results{RunID: "run-head", Repo: runner.RepoMetadata{Commit: "head"}, Summary: runner.RunSummary{PassRate: 0.7, TokensTotal: 20}}, "", nil
	}
	t.Cleanup(func() { resolveRun = origResolve })

	cmd := findCommand("compare")
	if cmd == nil {
		t.Fatalf("compare command not found")
	}
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{"--input", "/tmp/out", "--base", "base", "--head", "head"}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Delta")) {
		t.Fatalf("expected compare output")
	}
}

func TestReportCommand(t *testing.T) {
	origResolve := resolveRun
	origParseRange := parseRange
	origResolveRange := resolveRange
	origBuildHTML := buildReportHTML
	t.Cleanup(func() {
		resolveRun = origResolve
		parseRange = origParseRange
		resolveRange = origResolveRange
		buildReportHTML = origBuildHTML
	})

	parseRange = func(_ string) (vcs.RangeSpec, error) {
		return vcs.RangeSpec{Start: "a", End: "b"}, nil
	}
	resolveRange = func(_ context.Context, _ string, _ vcs.RangeSpec) (vcs.RangeResult, error) {
		return vcs.RangeResult{Start: "a", End: "b", Commits: []string{"b"}}, nil
	}
	resolveRun = func(_ string, _ string, ref string) (runner.Results, string, error) {
		return runner.Results{RunID: "run-" + ref, Repo: runner.RepoMetadata{Commit: ref}}, "", nil
	}
	buildReportHTML = func(_ []runner.Results) string {
		return "<html>report</html>"
	}

	cmd := findCommand("report")
	if cmd == nil {
		t.Fatalf("report command not found")
	}
	reportPath := filepath.Join(t.TempDir(), "report.html")
	var stdout, stderr bytes.Buffer
	exitCode := cmd.Run([]string{"--input", "/tmp/out", "--range", "a..b", "--output", reportPath}, &stdout, &stderr)
	if exitCode != ExitOK {
		t.Fatalf("unexpected exit: %d, stderr: %s", exitCode, stderr.String())
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if string(data) != "<html>report</html>" {
		t.Fatalf("unexpected report output: %s", string(data))
	}
}
