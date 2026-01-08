package tools

import (
	"context"
	"fmt"
)

// ListDir executes the list_dir tool.
func (r *Runner) ListDir(ctx context.Context, args ListDirArgs) CallResult {
	start := r.clock()
	output, err := r.listDir(ctx, args)
	end := r.clock()
	return r.finalize("list_dir", start, end, output, false, err)
}

// listDir is the internal implementation of list_dir.
func (r *Runner) listDir(ctx context.Context, args ListDirArgs) (string, error) {
	_ = ctx
	_ = args
	return "", fmt.Errorf("list_dir is not implemented")
}
