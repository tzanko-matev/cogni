package tools

import (
	"context"
	"testing"

	"cogni/internal/testutil"
)

// recordingRGRunner captures the last rg arguments for assertions.
type recordingRGRunner struct {
	output   string
	err      error
	lastArgs []string
}

// Run satisfies rgRunner and records arguments for tests.
func (r *recordingRGRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	r.lastArgs = append([]string(nil), args...)
	return r.output, r.err
}

// TestListFilesGlobAllSkipsGlob verifies match-all globs don't override rg defaults.
func TestListFilesGlobAllSkipsGlob(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	recorder := &recordingRGRunner{output: "ok\n"}
	runner.rgRunner = recorder

	ctx := testutil.Context(t, 0)
	result := runner.ListFiles(ctx, ListFilesArgs{Glob: "*"})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if len(recorder.lastArgs) != 1 || recorder.lastArgs[0] != "--files" {
		t.Fatalf("expected only --files, got %v", recorder.lastArgs)
	}
}
