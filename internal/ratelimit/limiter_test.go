package ratelimit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogni/internal/spec"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
	"cogni/pkg/ratelimiter/local"
)

// TestBuildLimiterDisabledReturnsNoop ensures disabled mode returns a no-op limiter.
func TestBuildLimiterDisabledReturnsNoop(t *testing.T) {
	runWithTimeout(t, func() {
		cfg := spec.Config{RateLimiter: spec.RateLimiterConfig{Mode: "disabled"}}
		limiter, err := BuildLimiter(cfg, t.TempDir())
		if err != nil {
			t.Fatalf("build limiter: %v", err)
		}
		if limiter != ratelimiter.NoopLimiter {
			t.Fatalf("expected noop limiter")
		}
	})
}

// TestBuildLimiterEmbeddedLoadsLimits ensures embedded mode loads limits from a file.
func TestBuildLimiterEmbeddedLoadsLimits(t *testing.T) {
	runWithTimeout(t, func() {
		repoRoot := t.TempDir()
		limitsPath := writeLimitsFile(t, repoRoot, sampleLimitStates())
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:       "embedded",
				LimitsPath: filepath.Base(limitsPath),
				Batch:      spec.BatchConfig{Size: 1, FlushMs: 1},
			},
		}
		limiter, err := BuildLimiter(cfg, repoRoot)
		if err != nil {
			t.Fatalf("build limiter: %v", err)
		}
		if _, ok := limiter.(*local.Client); !ok {
			t.Fatalf("expected local limiter, got %T", limiter)
		}
	})
}

// TestBuildLimiterEmbeddedUsesInlineLimits ensures embedded mode can use inline limits.
func TestBuildLimiterEmbeddedUsesInlineLimits(t *testing.T) {
	runWithTimeout(t, func() {
		repoRoot := t.TempDir()
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:   "embedded",
				Limits: sampleLimitStates(),
				Batch:  spec.BatchConfig{Size: 1, FlushMs: 1},
			},
		}
		limiter, err := BuildLimiter(cfg, repoRoot)
		if err != nil {
			t.Fatalf("build limiter: %v", err)
		}
		if _, ok := limiter.(*local.Client); !ok {
			t.Fatalf("expected local limiter, got %T", limiter)
		}
	})
}

// TestBuildLimiterEmbeddedRejectsLimitsAndPath ensures embedded mode rejects both limits and limits_path.
func TestBuildLimiterEmbeddedRejectsLimitsAndPath(t *testing.T) {
	runWithTimeout(t, func() {
		repoRoot := t.TempDir()
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:       "embedded",
				Limits:     sampleLimitStates(),
				LimitsPath: "limits.json",
				Batch:      spec.BatchConfig{Size: 1, FlushMs: 1},
			},
		}
		_, err := BuildLimiter(cfg, repoRoot)
		if err == nil {
			t.Fatalf("expected error for embedded limits with path")
		}
	})
}

// TestBuildLimiterEmbeddedMissingFileFails ensures missing limits files return an error.
func TestBuildLimiterEmbeddedMissingFileFails(t *testing.T) {
	runWithTimeout(t, func() {
		repoRoot := t.TempDir()
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:       "embedded",
				LimitsPath: "missing.json",
				Batch:      spec.BatchConfig{Size: 1, FlushMs: 1},
			},
		}
		_, err := BuildLimiter(cfg, repoRoot)
		if err == nil {
			t.Fatalf("expected error for missing limits file")
		}
	})
}

// TestBuildLimiterRemoteConstructsClient ensures remote mode uses the HTTP client.
func TestBuildLimiterRemoteConstructsClient(t *testing.T) {
	runWithTimeout(t, func() {
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:             "remote",
				BaseURL:          "http://example",
				RequestTimeoutMs: 1234,
				Batch:            spec.BatchConfig{Size: 1, FlushMs: 1},
			},
		}
		limiter, err := BuildLimiter(cfg, t.TempDir())
		if err != nil {
			t.Fatalf("build limiter: %v", err)
		}
		if _, ok := limiter.(*httpclient.Client); !ok {
			t.Fatalf("expected http client limiter, got %T", limiter)
		}
	})
}

// TestBuildLimiterWrapsBatcher ensures batching wraps the limiter when configured.
func TestBuildLimiterWrapsBatcher(t *testing.T) {
	runWithTimeout(t, func() {
		cfg := spec.Config{
			RateLimiter: spec.RateLimiterConfig{
				Mode:             "remote",
				BaseURL:          "http://example",
				RequestTimeoutMs: 2000,
				Batch:            spec.BatchConfig{Size: 2, FlushMs: 1},
			},
		}
		limiter, err := BuildLimiter(cfg, t.TempDir())
		if err != nil {
			t.Fatalf("build limiter: %v", err)
		}
		if _, ok := limiter.(*ratelimiter.Batcher); !ok {
			t.Fatalf("expected batcher wrapper, got %T", limiter)
		}
	})
}

// runWithTimeout executes a test body with an explicit timeout.
func runWithTimeout(t *testing.T, fn func()) {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("test timed out")
	}
}

// writeLimitsFile writes a minimal limits JSON file and returns the path.
func writeLimitsFile(t *testing.T, dir string, states []ratelimiter.LimitState) string {
	t.Helper()
	payload, err := json.Marshal(states)
	if err != nil {
		t.Fatalf("marshal limits: %v", err)
	}
	path := filepath.Join(dir, "limits.json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatalf("write limits file: %v", err)
	}
	return path
}

func sampleLimitStates() []ratelimiter.LimitState {
	return []ratelimiter.LimitState{
		{
			Definition: ratelimiter.LimitDefinition{
				Key:            "global:llm:openrouter:model:concurrency",
				Kind:           ratelimiter.KindConcurrency,
				Capacity:       1,
				TimeoutSeconds: 60,
				Unit:           "requests",
				Overage:        ratelimiter.OverageDebt,
			},
			Status: ratelimiter.LimitStatusActive,
		},
	}
}
