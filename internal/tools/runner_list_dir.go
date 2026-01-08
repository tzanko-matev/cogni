package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
)

const (
	listDirDefaultOffset = 1
	listDirDefaultLimit  = 25
	listDirDefaultDepth  = 2
	listDirMaxNameBytes  = 500
)

// listDirEntry captures a formatted entry and its sort metadata.
type listDirEntry struct {
	normalized  string
	depth       int
	displayName string
}

// listDirNode tracks traversal state for a directory during BFS.
type listDirNode struct {
	abs        string
	normalized string
	depth      int
}

// ListDir executes the list_dir tool.
func (r *Runner) ListDir(ctx context.Context, args ListDirArgs) CallResult {
	start := r.clock()
	output, err := r.listDir(ctx, args)
	end := r.clock()
	return r.finalize("list_dir", start, end, output, false, err)
}

// listDir is the internal implementation of list_dir.
func (r *Runner) listDir(ctx context.Context, args ListDirArgs) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	offset, limit, depth, err := normalizeListDirParams(args.Offset, args.Limit, args.Depth)
	if err != nil {
		return "", err
	}
	rel, abs, err := resolvePath(r.Root, args.Path)
	if err != nil {
		return "", err
	}
	info, err := r.fs.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", rel, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}
	entries, err := r.collectListDirEntries(ctx, abs, rel, depth)
	if err != nil {
		return "", err
	}
	return formatListDirOutput(abs, entries, offset, limit)
}

// normalizeListDirParams applies defaults and validates pagination inputs.
func normalizeListDirParams(offset, limit, depth *int) (int, int, int, error) {
	resolvedOffset := listDirDefaultOffset
	if offset != nil {
		if *offset < 1 {
			return 0, 0, 0, fmt.Errorf("offset must be >= 1")
		}
		resolvedOffset = *offset
	}
	resolvedLimit := listDirDefaultLimit
	if limit != nil {
		if *limit < 1 {
			return 0, 0, 0, fmt.Errorf("limit must be >= 1")
		}
		resolvedLimit = *limit
	}
	resolvedDepth := listDirDefaultDepth
	if depth != nil {
		if *depth < 1 {
			return 0, 0, 0, fmt.Errorf("depth must be >= 1")
		}
		resolvedDepth = *depth
	}
	return resolvedOffset, resolvedLimit, resolvedDepth, nil
}

// collectListDirEntries gathers entries in BFS order with per-directory sorting.
func (r *Runner) collectListDirEntries(ctx context.Context, absRoot, relRoot string, maxDepth int) ([]listDirEntry, error) {
	queue := []listDirNode{{abs: absRoot, normalized: "", depth: 1}}
	entries := make([]listDirEntry, 0, listDirDefaultLimit)
	for len(queue) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		node := queue[0]
		queue = queue[1:]
		dirEntries, err := r.fs.ReadDir(node.abs)
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", listDirRelPath(relRoot, node.normalized), err)
		}
		batch := make([]listDirEntry, 0, len(dirEntries))
		for _, entry := range dirEntries {
			name := entry.Name()
			absEntry := filepath.Join(node.abs, name)
			info, err := r.fs.Lstat(absEntry)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", filepath.Join(listDirRelPath(relRoot, node.normalized), name), err)
			}
			suffix, isDir, isSymlink := classifyListDirEntry(info)
			normalized := joinNormalizedPath(node.normalized, name)
			batch = append(batch, listDirEntry{
				normalized:  normalized,
				depth:       node.depth,
				displayName: truncateListDirName(name) + suffix,
			})
			if isDir && !isSymlink && node.depth < maxDepth {
				queue = append(queue, listDirNode{abs: absEntry, normalized: normalized, depth: node.depth + 1})
			}
		}
		sort.Slice(batch, func(i, j int) bool {
			return batch[i].normalized < batch[j].normalized
		})
		entries = append(entries, batch...)
	}
	return entries, nil
}
