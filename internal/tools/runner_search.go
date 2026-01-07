package tools

import (
	"context"
	"fmt"
	"strings"
)

// Search executes the search tool.
func (r *Runner) Search(ctx context.Context, args SearchArgs) CallResult {
	start := r.clock()
	output, err := r.search(ctx, args)
	end := r.clock()
	return r.finalize("search", start, end, output, false, err)
}

// search runs ripgrep to find query matches.
func (r *Runner) search(ctx context.Context, args SearchArgs) (string, error) {
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	paths := make([]string, 0, len(args.Paths))
	for _, path := range args.Paths {
		rel, _, err := resolvePath(r.Root, path)
		if err != nil {
			return "", err
		}
		paths = append(paths, rel)
	}
	rgArgs := []string{"--no-heading", "--line-number", "--color", "never", "--", query}
	if len(paths) > 0 {
		rgArgs = append(rgArgs, paths...)
	}
	return runRG(ctx, r.Root, rgArgs...)
}
