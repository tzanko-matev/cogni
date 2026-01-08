package tools

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

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
