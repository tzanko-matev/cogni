package vcs

import (
	"context"
	"fmt"
	"strings"
)

type RangeSpec struct {
	Start string
	End   string
}

type RangeResult struct {
	Start   string
	End     string
	Commits []string
}

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

func ResolveRef(ctx context.Context, repoRoot, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", fmt.Errorf("ref is empty")
	}
	commit, err := runGit(ctx, repoRoot, "rev-parse", "--verify", ref)
	if err != nil {
		return "", fmt.Errorf("resolve ref %q: %w", ref, err)
	}
	return commit, nil
}

func ResolveRange(ctx context.Context, repoRoot string, spec RangeSpec) (RangeResult, error) {
	start, err := ResolveRef(ctx, repoRoot, spec.Start)
	if err != nil {
		return RangeResult{}, err
	}
	end, err := ResolveRef(ctx, repoRoot, spec.End)
	if err != nil {
		return RangeResult{}, err
	}
	revList, err := runGit(ctx, repoRoot, "rev-list", "--reverse", fmt.Sprintf("%s..%s", start, end))
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
