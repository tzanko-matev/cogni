package config

import (
	"strings"

	"cogni/internal/spec"
)

// Normalize fills defaults and propagates agent assignments.
func Normalize(cfg *spec.Config) {
	normalizeRateLimiter(cfg)
	if cfg.DefaultAgent == "" && len(cfg.Agents) == 1 {
		cfg.DefaultAgent = cfg.Agents[0].ID
	}
	for i := range cfg.Tasks {
		if cfg.Tasks[i].Agent == "" {
			cfg.Tasks[i].Agent = cfg.DefaultAgent
		}
	}
}

// normalizeRateLimiter fills defaults for the rate limiter config.
func normalizeRateLimiter(cfg *spec.Config) {
	if strings.TrimSpace(cfg.RateLimiter.Mode) == "" {
		cfg.RateLimiter.Mode = "disabled"
	}
	if cfg.RateLimiter.Workers == 0 {
		cfg.RateLimiter.Workers = 1
	}
	if cfg.RateLimiter.RequestTimeoutMs == 0 {
		cfg.RateLimiter.RequestTimeoutMs = 2000
	}
	if cfg.RateLimiter.MaxOutputTokens == 0 {
		cfg.RateLimiter.MaxOutputTokens = 2048
	}
	if cfg.RateLimiter.Batch.Size == 0 {
		cfg.RateLimiter.Batch.Size = 128
	}
	if cfg.RateLimiter.Batch.FlushMs == 0 {
		cfg.RateLimiter.Batch.FlushMs = 2
	}
}
