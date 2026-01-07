package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config path constants used by the CLI and loaders.
const (
	ConfigDirName    = ".cogni"
	ConfigFileName   = "config.yml"
	DefaultOutputDir = ".cogni/results"
)

// ConfigDir returns the .cogni directory under the repo root.
func ConfigDir(root string) string {
	return filepath.Join(root, ConfigDirName)
}

// ConfigPath returns the full config file path under the repo root.
func ConfigPath(root string) string {
	return filepath.Join(ConfigDir(root), ConfigFileName)
}

// RepoRootFromConfigPath derives the repo root from a config file path.
func RepoRootFromConfigPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if filepath.Base(dir) == ConfigDirName {
		return filepath.Dir(dir)
	}
	return dir
}

// FindConfigPath searches upward from a directory for a config file.
func FindConfigPath(startDir string) (string, error) {
	dir := strings.TrimSpace(startDir)
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		dir = wd
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve start directory: %w", err)
	}
	dir = abs

	for {
		configDir := filepath.Join(dir, ConfigDirName)
		configPath := filepath.Join(configDir, ConfigFileName)
		info, err := os.Stat(configPath)
		if err == nil {
			if info.IsDir() {
				return "", fmt.Errorf("config path %q is a directory", configPath)
			}
			return configPath, nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("stat config path %q: %w", configPath, err)
		}
		if dirInfo, dirErr := os.Stat(configDir); dirErr == nil && dirInfo.IsDir() {
			return "", fmt.Errorf("found %q but %s is missing", configDir, ConfigFileName)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s found in %s or parent directories", filepath.Join(ConfigDirName, ConfigFileName), dir)
		}
		dir = parent
	}
}
