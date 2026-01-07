package config

import (
	"fmt"
	"strings"

	"cogni/internal/spec"
)

// Validate checks a config for correctness and referenced files.
func Validate(cfg *spec.Config, baseDir string) error {
	collector := &issueCollector{}

	if cfg.Version == 0 {
		collector.add("version", "is required")
	} else if cfg.Version != 1 {
		collector.add("version", fmt.Sprintf("unsupported version %d", cfg.Version))
	}

	if strings.TrimSpace(cfg.Repo.OutputDir) == "" {
		collector.add("repo.output_dir", "is required")
	}

	if baseDir == "" {
		baseDir = "."
	}

	agentIDs := validateAgents(cfg, collector.add)
	validateDefaultAgent(cfg, agentIDs, collector.add)
	adapterIDs := validateAdapters(cfg, baseDir, collector.add)
	validateTasks(cfg, baseDir, agentIDs, adapterIDs, collector.add)

	return collector.result()
}
