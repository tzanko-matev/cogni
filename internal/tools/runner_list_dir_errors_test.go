package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

func TestListDirSymlinkNotTraversed(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "real"))
	mustWriteFile(t, filepath.Join(root, "real", "child.txt"), "child")
	linkPath := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "real"), linkPath); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	ctx := testutil.Context(t, 0)
	depth := 2
	result := runner.ListDir(ctx, ListDirArgs{Path: ".", Depth: &depth})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if !strings.Contains(result.Output, "link@") {
		t.Fatalf("expected symlink entry, got %q", result.Output)
	}
	if strings.Count(result.Output, "child.txt") != 1 {
		t.Fatalf("expected only real child entry, got %q", result.Output)
	}
}

func TestListDirErrors(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "file.txt"), "data")
	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	ctx := testutil.Context(t, 0)

	result := runner.ListDir(ctx, ListDirArgs{Path: "../outside"})
	if result.Error == "" || !strings.Contains(result.Error, "escapes root") {
		t.Fatalf("expected escape error, got %q", result.Error)
	}

	result = runner.ListDir(ctx, ListDirArgs{Path: "file.txt"})
	if result.Error != "path is not a directory" {
		t.Fatalf("expected non-dir error, got %q", result.Error)
	}

	offset := 0
	result = runner.ListDir(ctx, ListDirArgs{Path: ".", Offset: &offset})
	if result.Error != "offset must be >= 1" {
		t.Fatalf("expected offset error, got %q", result.Error)
	}

	limit := 0
	result = runner.ListDir(ctx, ListDirArgs{Path: ".", Limit: &limit})
	if result.Error != "limit must be >= 1" {
		t.Fatalf("expected limit error, got %q", result.Error)
	}

	depth := 0
	result = runner.ListDir(ctx, ListDirArgs{Path: ".", Depth: &depth})
	if result.Error != "depth must be >= 1" {
		t.Fatalf("expected depth error, got %q", result.Error)
	}

	offset = 5
	result = runner.ListDir(ctx, ListDirArgs{Path: ".", Offset: &offset})
	if result.Error != "offset exceeds directory entry count" {
		t.Fatalf("expected offset count error, got %q", result.Error)
	}
}
