package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"cogni/internal/config"
)

// resolveSpecPath normalizes a config path or finds it from CWD.
func resolveSpecPath(specPath string) (string, error) {
	if strings.TrimSpace(specPath) == "" {
		return config.FindConfigPath("")
	}
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return "", fmt.Errorf("resolve spec path: %w", err)
	}
	return abs, nil
}
