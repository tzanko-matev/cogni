package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

func TestListDirDepthAndIndentation(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "nested", "deeper"))
	mustWriteFile(t, filepath.Join(root, "root.txt"), "root")
	mustWriteFile(t, filepath.Join(root, "nested", "child.txt"), "child")
	mustWriteFile(t, filepath.Join(root, "nested", "deeper", "grandchild.txt"), "grand")

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	ctx := testutil.Context(t, 0)

	cases := []struct {
		name  string
		depth int
		lines []string
	}{
		{
			name:  "depth1",
			depth: 1,
			lines: []string{
				"Absolute path: " + root,
				"nested/",
				"root.txt",
			},
		},
		{
			name:  "depth2",
			depth: 2,
			lines: []string{
				"Absolute path: " + root,
				"nested/",
				"  child.txt",
				"  deeper/",
				"root.txt",
			},
		},
		{
			name:  "depth3",
			depth: 3,
			lines: []string{
				"Absolute path: " + root,
				"nested/",
				"  child.txt",
				"  deeper/",
				"    grandchild.txt",
				"root.txt",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			depth := tc.depth
			result := runner.ListDir(ctx, ListDirArgs{Path: ".", Depth: &depth})
			if result.Error != "" {
				t.Fatalf("unexpected error: %v", result.Error)
			}
			lines := strings.Split(result.Output, "\n")
			if !reflect.DeepEqual(lines, tc.lines) {
				t.Fatalf("unexpected output:\n%v", lines)
			}
		})
	}
}

func TestListDirPaginationAndMoreMarker(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll(t, filepath.Join(root, "a_dir"))
	mustWriteFile(t, filepath.Join(root, "a_dir", "z.txt"), "z")
	mustWriteFile(t, filepath.Join(root, "b.txt"), "b")
	mustWriteFile(t, filepath.Join(root, "c.txt"), "c")

	runner, err := NewRunner(root)
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}
	ctx := testutil.Context(t, 0)

	offset := 1
	limit := 2
	result := runner.ListDir(ctx, ListDirArgs{Path: ".", Offset: &offset, Limit: &limit})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	lines := strings.Split(result.Output, "\n")
	expected := []string{
		"Absolute path: " + root,
		"a_dir/",
		"b.txt",
		"More than 2 entries found.",
	}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("unexpected output:\n%v", lines)
	}

	offset = 3
	limit = 2
	result = runner.ListDir(ctx, ListDirArgs{Path: ".", Offset: &offset, Limit: &limit})
	if result.Error != "" {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	lines = strings.Split(result.Output, "\n")
	expected = []string{
		"Absolute path: " + root,
		"  z.txt",
		"c.txt",
	}
	if !reflect.DeepEqual(lines, expected) {
		t.Fatalf("unexpected output:\n%v", lines)
	}
}

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

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
}
