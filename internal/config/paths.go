package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ConfigDirName    = ".cogni"
	ConfigFileName   = "config.yml"
	DefaultOutputDir = ".cogni/results"
)

func ConfigDir(root string) string {
	return filepath.Join(root, ConfigDirName)
}

func ConfigPath(root string) string {
	return filepath.Join(ConfigDir(root), ConfigFileName)
}

func RepoRootFromConfigPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if filepath.Base(dir) == ConfigDirName {
		return filepath.Dir(dir)
	}
	return dir
}

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
