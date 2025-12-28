package config

import "cogni/internal/spec"

func Normalize(cfg *spec.Config) {
	if cfg.DefaultAgent == "" && len(cfg.Agents) == 1 {
		cfg.DefaultAgent = cfg.Agents[0].ID
	}
	for i := range cfg.Tasks {
		if cfg.Tasks[i].Agent == "" {
			cfg.Tasks[i].Agent = cfg.DefaultAgent
		}
	}
}
