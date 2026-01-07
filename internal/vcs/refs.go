package vcs

import (
	"context"
	"fmt"
	"strings"
)

// RangeSpec identifies a start and end ref.
type RangeSpec struct {
	Start string
	End   string
}

// RangeResult contains resolved refs and commit list.
type RangeResult struct {
	Start   string
	End     string
	Commits []string
}

// ParseRange parses a range spec in the form start..end.
func ParseRange(spec string) (RangeSpec, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return RangeSpec{}, fmt.Errorf("range is empty")
	}
	if strings.Contains(trimmed, "...") {
		return RangeSpec{}, fmt.Errorf("range must use '..' not '...'")
	}
	if strings.Count(trimmed, "..") != 1 {
		return RangeSpec{}, fmt.Errorf("range must contain a single '..'")
	}
	parts := strings.SplitN(trimmed, "..", 2)
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if start == "" || end == "" {
		return RangeSpec{}, fmt.Errorf("range must include start and end")
	}
	return RangeSpec{
		Start: start,
		End:   end,
	}, nil
}

// ResolveRef resolves a git ref to a commit hash.
func ResolveRef(ctx context.Context, repoRoot, ref string) (string, error) {
	return defaultClient.ResolveRef(ctx, repoRoot, ref)
}

// ResolveRange resolves a RangeSpec into commits between refs.
func ResolveRange(ctx context.Context, repoRoot string, spec RangeSpec) (RangeResult, error) {
	return defaultClient.ResolveRange(ctx, repoRoot, spec)
}

// ResolveRef resolves a git ref to a commit hash using a client runner.
func (c Client) ResolveRef(ctx context.Context, repoRoot, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", fmt.Errorf("ref is empty")
	}
	commit, err := c.runner.Run(ctx, repoRoot, "rev-parse", "--verify", ref)
	if err != nil {
		return "", fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	return commit, nil
}

// ResolveRange resolves a RangeSpec into commits between refs using a client runner.
func (c Client) ResolveRange(ctx context.Context, repoRoot string, spec RangeSpec) (RangeResult, error) {
	start, err := c.ResolveRef(ctx, repoRoot, spec.Start)
	if err != nil {
		return RangeResult{}, err
	}
	end, err := c.ResolveRef(ctx, repoRoot, spec.End)
	if err != nil {
		return RangeResult{}, err
	}
	revList, err := c.runner.Run(ctx, repoRoot, "rev-list", "--reverse", fmt.Sprintf("%s..%s", start, end))
	if err != nil {
		return RangeResult{}, fmt.Errorf("resolve range %s..%s: %w", spec.Start, spec.End, err)
	}
	commits := []string{}
	if strings.TrimSpace(revList) != "" {
		commits = strings.Split(revList, "\n")
	}
	return RangeResult{
		Start:   start,
		End:     end,
		Commits: commits,
	}, nil
}
