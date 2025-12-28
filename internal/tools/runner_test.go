package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
	result := runner.ReadFile(context.Background(), ReadFileArgs{
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

func TestReadFileOutsideRoot(t *testing.T) {
	root := t.TempDir()
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	result := runner.ReadFile(context.Background(), ReadFileArgs{
		Path: "../outside.txt",
	})
	if result.Error == "" {
		t.Fatalf("expected error")
	}
	if !strings.HasPrefix(result.Output, "error:") {
		t.Fatalf("expected error output")
	}
}

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
	result := runner.ReadFile(context.Background(), ReadFileArgs{
		Path:      "sample.txt",
		StartLine: &start,
	})
	if result.Error == "" {
		t.Fatalf("expected error")
	}
}

func TestSearchNoMatches(t *testing.T) {
	requireRG(t)

	root := t.TempDir()
	path := filepath.Join(root, "sample.txt")
	if err := os.WriteFile(path, []byte("hello world\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	result := runner.Search(context.Background(), SearchArgs{
		Query: "missing",
	})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if result.Output != "" {
		t.Fatalf("expected empty output, got %q", result.Output)
	}
}

func TestSearchMatchesAndTruncation(t *testing.T) {
	requireRG(t)

	root := t.TempDir()
	path := filepath.Join(root, "sample.txt")
	content := "long " + strings.Repeat("x", 200) + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	runner.Limits.MaxOutputBytes = 64
	result := runner.Search(context.Background(), SearchArgs{
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

func TestListFilesGlob(t *testing.T) {
	requireRG(t)

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	result := runner.ListFiles(context.Background(), ListFilesArgs{
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

func requireRG(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg not available")
	}
}
