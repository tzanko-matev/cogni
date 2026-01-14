package config

import (
	"strings"

	"cogni/internal/spec"
)

// validateRateLimiter checks rate limiter configuration rules.
func validateRateLimiter(cfg *spec.Config, add issueAdder) {
	mode := strings.ToLower(strings.TrimSpace(cfg.RateLimiter.Mode))
	switch mode {
	case "", "disabled":
		// No additional fields required.
	case "remote":
		if strings.TrimSpace(cfg.RateLimiter.BaseURL) == "" {
			add("rate_limiter.base_url", "is required when mode is remote")
		}
	case "embedded":
		limitsPath := strings.TrimSpace(cfg.RateLimiter.LimitsPath)
		limitsProvided := cfg.RateLimiter.Limits != nil
		switch {
		case limitsPath == "" && !limitsProvided:
			add("rate_limiter.limits", "or limits_path is required when mode is embedded")
		case limitsPath != "" && limitsProvided:
			add("rate_limiter.limits", "cannot be set with limits_path when mode is embedded")
		}
	default:
		add("rate_limiter.mode", "must be one of disabled, remote, embedded")
	}

	if cfg.RateLimiter.Workers < 1 {
		add("rate_limiter.workers", "must be >= 1")
	}
	if cfg.RateLimiter.Batch.Size < 1 {
		add("rate_limiter.batch.size", "must be >= 1")
	}
	if cfg.RateLimiter.Batch.FlushMs < 1 {
		add("rate_limiter.batch.flush_ms", "must be >= 1")
	}
	if cfg.RateLimiter.RequestTimeoutMs < 1 {
		add("rate_limiter.request_timeout_ms", "must be >= 1")
	}
}
