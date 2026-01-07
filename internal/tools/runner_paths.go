package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolvePath resolves a relative path within the repo root.
func resolvePath(root, path string) (string, string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", "", fmt.Errorf("path is empty")
	}
	cleaned := filepath.Clean(trimmed)
	var rel string
	if filepath.IsAbs(cleaned) {
		relative, err := filepath.Rel(root, cleaned)
		if err != nil {
			return "", "", fmt.Errorf("resolve path %q: %w", path, err)
		}
		rel = relative
	} else {
		rel = cleaned
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", "", fmt.Errorf("path %q escapes root", path)
	}
	abs := filepath.Join(root, rel)
	return rel, abs, nil
}
