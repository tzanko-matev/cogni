package vcs

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

// TestDiscoverRepoRootAndMetadata verifies repo discovery and metadata parsing.
func TestDiscoverRepoRootAndMetadata(t *testing.T) {
	ctx := testutil.Context(t, 0)
	root := filepath.Join(t.TempDir(), "repo")
	subdir := filepath.Join(root, "nested")

	fake := &fakeGitRunner{responses: map[string]string{
		"rev-parse --show-toplevel":   root,
		"rev-parse HEAD":              "commit-3",
		"rev-parse --abbrev-ref HEAD": "main",
		"status --porcelain":          "",
	}}
	client := NewClient(fake)

	actualRoot, err := client.DiscoverRepoRoot(ctx, subdir)
	if err != nil {
		t.Fatalf("discover repo root: %v", err)
	}
	if actualRoot != root {
		t.Fatalf("expected root %q, got %q", root, actualRoot)
	}

	repo, err := client.Discover(ctx, subdir)
	if err != nil {
		t.Fatalf("discover repo: %v", err)
	}
	meta, err := repo.Metadata(ctx)
	if err != nil {
		t.Fatalf("metadata: %v", err)
	}
	if meta.Commit != "commit-3" {
		t.Fatalf("expected commit %q, got %q", "commit-3", meta.Commit)
	}
	if meta.Branch != "main" {
		t.Fatalf("expected branch main, got %q", meta.Branch)
	}
	if meta.Dirty {
		t.Fatalf("expected clean repo, got dirty")
	}
	if meta.Name != filepath.Base(root) {
		t.Fatalf("expected name %q, got %q", filepath.Base(root), meta.Name)
	}

	fake.responses["status --porcelain"] = " M README.md"
	meta, err = repo.Metadata(ctx)
	if err != nil {
		t.Fatalf("metadata dirty: %v", err)
	}
	if !meta.Dirty {
		t.Fatalf("expected dirty repo")
	}
}

// fakeGitRunner returns canned outputs for git commands in tests.
type fakeGitRunner struct {
	responses map[string]string
}

// Run satisfies gitRunner for test doubles.
func (f *fakeGitRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	key := strings.Join(args, " ")
	if value, ok := f.responses[key]; ok {
		return value, nil
	}
	return "", fmt.Errorf("unexpected git args: %s", key)
}
