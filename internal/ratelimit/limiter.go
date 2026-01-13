package ratelimit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cogni/internal/spec"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
	"cogni/pkg/ratelimiter/local"
)

const fallbackMaxOutputTokens uint64 = 2048

// BuildLimiter constructs a limiter client based on configuration.
func BuildLimiter(cfg spec.Config, repoRoot string) (ratelimiter.Limiter, error) {
	mode := strings.ToLower(strings.TrimSpace(cfg.RateLimiter.Mode))
	switch mode {
	case "", "disabled":
		return ratelimiter.NoopLimiter, nil
	case "remote":
		return buildRemoteLimiter(cfg)
	case "embedded":
		return buildEmbeddedLimiter(cfg, repoRoot)
	default:
		return nil, fmt.Errorf("unsupported rate limiter mode %q", cfg.RateLimiter.Mode)
	}
}

// ResolveTaskWorkers returns the worker count for a task run.
func ResolveTaskWorkers(cfg spec.Config, task spec.TaskConfig) int {
	if task.Concurrency > 0 {
		return task.Concurrency
	}
	if cfg.RateLimiter.Workers > 0 {
		return cfg.RateLimiter.Workers
	}
	return 1
}

// MaxOutputTokens returns the max output token limit for reservations.
func MaxOutputTokens(cfg spec.Config, task spec.TaskConfig) uint64 {
	if task.Budget.MaxTokens > 0 {
		return uint64(task.Budget.MaxTokens)
	}
	if cfg.RateLimiter.MaxOutputTokens > 0 {
		return cfg.RateLimiter.MaxOutputTokens
	}
	return fallbackMaxOutputTokens
}

// buildRemoteLimiter constructs an HTTP limiter client and wraps batching when enabled.
func buildRemoteLimiter(cfg spec.Config) (ratelimiter.Limiter, error) {
	if strings.TrimSpace(cfg.RateLimiter.BaseURL) == "" {
		return nil, fmt.Errorf("rate limiter base_url is required for remote mode")
	}
	timeout := time.Duration(cfg.RateLimiter.RequestTimeoutMs) * time.Millisecond
	limiter := ratelimiter.Limiter(httpclient.NewWithTimeout(cfg.RateLimiter.BaseURL, timeout))
	return wrapBatcher(cfg, limiter), nil
}

// buildEmbeddedLimiter constructs an in-memory limiter client and wraps batching when enabled.
func buildEmbeddedLimiter(cfg spec.Config, repoRoot string) (ratelimiter.Limiter, error) {
	if strings.TrimSpace(cfg.RateLimiter.LimitsPath) == "" {
		return nil, fmt.Errorf("rate limiter limits_path is required for embedded mode")
	}
	limitsPath := resolveLimitsPath(repoRoot, cfg.RateLimiter.LimitsPath)
	if _, err := os.Stat(limitsPath); err != nil {
		return nil, fmt.Errorf("read limits file: %w", err)
	}
	limiter, err := local.NewMemoryLimiterFromFile(limitsPath)
	if err != nil {
		return nil, err
	}
	return wrapBatcher(cfg, limiter), nil
}

// resolveLimitsPath resolves a limits file path against the repository root.
func resolveLimitsPath(repoRoot, limitsPath string) string {
	if filepath.IsAbs(limitsPath) || strings.TrimSpace(repoRoot) == "" {
		return limitsPath
	}
	return filepath.Join(repoRoot, limitsPath)
}

// wrapBatcher wraps the limiter with a batcher when configured.
func wrapBatcher(cfg spec.Config, limiter ratelimiter.Limiter) ratelimiter.Limiter {
	if cfg.RateLimiter.Batch.Size > 1 {
		flush := time.Duration(cfg.RateLimiter.Batch.FlushMs) * time.Millisecond
		return ratelimiter.NewBatcher(limiter, cfg.RateLimiter.Batch.Size, flush)
	}
	return limiter
}
