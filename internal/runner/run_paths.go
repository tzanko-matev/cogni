package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/vcs"
)

// resolveRepoRoot resolves the repository root for a run.
func resolveRepoRoot(ctx context.Context, repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		repoRoot = wd
	}
	return vcs.DiscoverRepoRoot(ctx, repoRoot)
}

// resolveOutputDir resolves relative output paths against the repo root.
func resolveOutputDir(repoRoot, outputDir string) string {
	if outputDir == "" || filepath.IsAbs(outputDir) {
		return outputDir
	}
	return filepath.Join(repoRoot, outputDir)
}

// loadRepoMetadata loads VCS metadata for the repo.
func loadRepoMetadata(ctx context.Context, repoRoot string) (vcs.Metadata, error) {
	repo, err := vcs.Discover(ctx, repoRoot)
	if err != nil {
		return vcs.Metadata{}, err
	}
	return repo.Metadata(ctx)
}

// ensureRunID uses the provided generator or falls back to NewRunID.
func ensureRunID(generator func() (string, error)) (string, error) {
	if generator != nil {
		return generator()
	}
	return NewRunID()
}
