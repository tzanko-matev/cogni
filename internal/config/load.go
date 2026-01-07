package config

import (
	"fmt"
	"os"

	"cogni/internal/spec"
)

// Load reads, parses, normalizes, and validates a config file.
func Load(path string) (spec.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return spec.Config{}, fmt.Errorf("read config: %w", err)
	}
	cfg, err := spec.ParseConfig(data)
	if err != nil {
		return spec.Config{}, err
	}
	Normalize(&cfg)
	if err := Validate(&cfg, RepoRootFromConfigPath(path)); err != nil {
		return spec.Config{}, err
	}
	return cfg, nil
}
