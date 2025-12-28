package vcs

import (
	"context"
	"testing"
)

func TestParseRange(t *testing.T) {
	spec, err := ParseRange("main..HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Start != "main" || spec.End != "HEAD" {
		t.Fatalf("unexpected range: %+v", spec)
	}
}

func TestParseRangeErrors(t *testing.T) {
	cases := []string{
		"",
		"main",
		"..HEAD",
		"main..",
		"main...HEAD",
	}
	for _, input := range cases {
		if _, err := ParseRange(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestResolveRefAndRange(t *testing.T) {
	requireGit(t)

	repo := setupTestRepo(t)
	ctx := context.Background()

	head, err := ResolveRef(ctx, repo.Root, "HEAD")
	if err != nil {
		t.Fatalf("resolve ref: %v", err)
	}
	if head != repo.Commits[len(repo.Commits)-1] {
		t.Fatalf("expected head %q, got %q", repo.Commits[len(repo.Commits)-1], head)
	}

	rangeSpec := RangeSpec{
		Start: repo.Commits[0],
		End:   repo.Commits[2],
	}
	resolved, err := ResolveRange(ctx, repo.Root, rangeSpec)
	if err != nil {
		t.Fatalf("resolve range: %v", err)
	}
	if resolved.Start != repo.Commits[0] || resolved.End != repo.Commits[2] {
		t.Fatalf("unexpected range: %+v", resolved)
	}
	if len(resolved.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(resolved.Commits))
	}
	if resolved.Commits[0] != repo.Commits[1] || resolved.Commits[1] != repo.Commits[2] {
		t.Fatalf("unexpected commits: %+v", resolved.Commits)
	}
}
