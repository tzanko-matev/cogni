package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

// fakeRGRunner returns canned outputs for ripgrep calls in tests.
type fakeRGRunner struct {
	output string
	err    error
}

// Run satisfies rgRunner for test doubles.
func (f fakeRGRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	return f.output, f.err
}

// TestReadFileRangeAndTruncation verifies read ranges and truncation logic.
func TestReadFileRangeAndTruncation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.txt")
	var builder strings.Builder
	for i := 1; i <= 10; i++ {
		builder.WriteString("line")
		builder.WriteString(strings.Repeat("x", i))
		builder.WriteString("\n")
	}
	if err := os.WriteFile(path, []byte(builder.String()), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	runner.Limits.MaxReadBytes = 40
	runner.Limits.MaxOutputBytes = 80

	start := 1
	end := 10
	ctx := testutil.Context(t, 0)
	result := runner.ReadFile(ctx, ReadFileArgs{
		Path:      "sample.txt",
		StartLine: &start,
		EndLine:   &end,
	})

	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !result.Truncated {
		t.Fatalf("expected truncated output")
	}
	if !strings.Contains(result.Output, "sample.txt") {
		t.Fatalf("expected output to include path header")
	}
	if !strings.Contains(result.Output, "truncated") {
		t.Fatalf("expected truncation marker")
	}
}

// TestReadFileOutsideRoot verifies escaping paths are rejected.
func TestReadFileOutsideRoot(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	ctx := testutil.Context(t, 0)
	result := runner.ReadFile(ctx, ReadFileArgs{
		Path: "../outside.txt",
	})
	if result.Error == "" {
		t.Fatalf("expected error")
	}
	if !strings.HasPrefix(result.Output, "error:") {
		t.Fatalf("expected error output")
	}
}

// TestReadFileInvalidRange verifies invalid line ranges are rejected.
func TestReadFileInvalidRange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	start := 0
	ctx := testutil.Context(t, 0)
	result := runner.ReadFile(ctx, ReadFileArgs{
		Path:      "sample.txt",
		StartLine: &start,
	})
	if result.Error == "" {
		t.Fatalf("expected error")
	}
}

// TestSearchNoMatches verifies empty search results return clean output.
func TestSearchNoMatches(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	runner.rgRunner = fakeRGRunner{output: ""}
	ctx := testutil.Context(t, 0)
	result := runner.Search(ctx, SearchArgs{
		Query: "missing",
	})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Output != "" {
		t.Fatalf("expected empty output, got %q", result.Output)
	}
}

// TestSearchMatchesAndTruncation verifies search output truncation.
func TestSearchMatchesAndTruncation(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	output := "sample.txt:1:long " + strings.Repeat("x", 200) + "\n"
	runner.rgRunner = fakeRGRunner{output: output}
	runner.Limits.MaxOutputBytes = 64
	ctx := testutil.Context(t, 0)
	result := runner.Search(ctx, SearchArgs{
		Query: "long",
	})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "sample.txt") {
		t.Fatalf("expected output to mention file")
	}
	if !result.Truncated {
		t.Fatalf("expected truncated output")
	}
}

// TestListFilesGlob verifies glob filtering for list_files.
func TestListFilesGlob(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	runner.rgRunner = fakeRGRunner{output: "b.go\n"}
	ctx := testutil.Context(t, 0)
	result := runner.ListFiles(ctx, ListFilesArgs{
		Glob: "*.go",
	})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "b.go") {
		t.Fatalf("expected output to include b.go, got %q", result.Output)
	}
	if strings.Contains(result.Output, "a.txt") {
		t.Fatalf("expected output to exclude a.txt, got %q", result.Output)
	}
}
