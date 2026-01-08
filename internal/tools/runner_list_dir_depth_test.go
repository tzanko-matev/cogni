package tools

import (
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
