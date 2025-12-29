package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func addGitignoreEntry(repoRoot, outputDir string) (bool, error) {
	entry, err := normalizeGitignorePath(repoRoot, outputDir)
	if err != nil {
		return false, err
	}
	if entry == "" {
		return false, fmt.Errorf("gitignore entry is empty")
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	var existing []byte
	if data, err := os.ReadFile(gitignorePath); err == nil {
		existing = data
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("read .gitignore: %w", err)
	}

	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == entry {
			return false, nil
		}
	}

	updated := string(existing)
	if len(updated) > 0 && !strings.HasSuffix(updated, "\n") {
		updated += "\n"
	}
	updated += entry + "\n"
	if err := os.WriteFile(gitignorePath, []byte(updated), 0o644); err != nil {
		return false, fmt.Errorf("write .gitignore: %w", err)
	}
	return true, nil
}

func normalizeGitignorePath(repoRoot, outputDir string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("output dir is required")
	}
	clean := filepath.Clean(outputDir)
	if filepath.IsAbs(clean) {
		rel, err := filepath.Rel(repoRoot, clean)
		if err != nil {
			return "", fmt.Errorf("resolve output dir: %w", err)
		}
		if strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("output dir %q is outside the repo root", outputDir)
		}
		clean = rel
	}
	clean = strings.TrimPrefix(clean, "."+string(filepath.Separator))
	if clean == "." || clean == "" || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("output dir %q is outside the repo root", outputDir)
	}
	return filepath.ToSlash(clean), nil
}
