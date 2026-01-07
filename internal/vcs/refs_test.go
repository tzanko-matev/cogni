package vcs

import (
	"testing"

	"cogni/internal/testutil"
)

// TestParseRange verifies valid range parsing.
func TestParseRange(t *testing.T) {
	spec, err := ParseRange("main..HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if spec.Start != "main" || spec.End != "HEAD" {
		t.Fatalf("unexpected range: %+v", spec)
	}
}

// TestParseRangeErrors verifies invalid range inputs error.
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

// TestResolveRefAndRange verifies ref and range resolution with a fake runner.
func TestResolveRefAndRange(t *testing.T) {
	ctx := testutil.Context(t, 0)
	fake := &fakeGitRunner{responses: map[string]string{
		"rev-parse --verify HEAD":               "commit-2",
		"rev-parse --verify base":               "commit-0",
		"rev-parse --verify head":               "commit-2",
		"rev-list --reverse commit-0..commit-2": "commit-1\ncommit-2",
	}}
	client := NewClient(fake)

	head, err := client.ResolveRef(ctx, "/repo", "HEAD")
	if err != nil {
		t.Fatalf("resolve ref: %v", err)
	}
	if head != "commit-2" {
		t.Fatalf("expected head %q, got %q", "commit-2", head)
	}

	rangeSpec := RangeSpec{
		Start: "base",
		End:   "head",
	}
	resolved, err := client.ResolveRange(ctx, "/repo", rangeSpec)
	if err != nil {
		t.Fatalf("resolve range: %v", err)
	}
	if resolved.Start != "commit-0" || resolved.End != "commit-2" {
		t.Fatalf("unexpected range: %+v", resolved)
	}
	if len(resolved.Commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(resolved.Commits))
	}
	if resolved.Commits[0] != "commit-1" || resolved.Commits[1] != "commit-2" {
		t.Fatalf("unexpected commits: %+v", resolved.Commits)
	}
}
