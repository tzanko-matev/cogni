package tools

import (
	"context"
	"strings"
)

// ListFiles executes the list_files tool.
func (r *Runner) ListFiles(ctx context.Context, args ListFilesArgs) CallResult {
	start := r.clock()
	output, err := r.listFiles(ctx, args)
	end := r.clock()
	return r.finalize("list_files", start, end, output, false, err)
}

// listFiles returns file listings using ripgrep.
func (r *Runner) listFiles(ctx context.Context, args ListFilesArgs) (string, error) {
	rgArgs := []string{"--files"}
	if glob := strings.TrimSpace(args.Glob); glob != "" {
		rgArgs = append(rgArgs, "-g", glob)
	}
	return runRG(ctx, r.Root, rgArgs...)
}
