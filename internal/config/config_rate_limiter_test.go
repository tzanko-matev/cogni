package config

import (
	"strings"
	"testing"
	"time"

	"cogni/internal/spec"
	"cogni/internal/testutil"
)

// TestValidateRateLimiterRemoteRequiresBaseURL ensures remote mode requires a base URL.
func TestValidateRateLimiterRemoteRequiresBaseURL(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Mode = "remote"
	cfg.RateLimiter.BaseURL = ""

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.base_url") {
		t.Fatalf("expected base_url error, got %q", err.Error())
	}
}

// TestValidateRateLimiterEmbeddedRequiresLimitsPath ensures embedded mode requires limits_path.
func TestValidateRateLimiterEmbeddedRequiresLimitsPath(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Mode = "embedded"
	cfg.RateLimiter.LimitsPath = ""

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.limits_path") {
		t.Fatalf("expected limits_path error, got %q", err.Error())
	}
}

// TestValidateRateLimiterRejectsInvalidMode ensures invalid modes are rejected.
func TestValidateRateLimiterRejectsInvalidMode(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Mode = "nope"

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.mode") {
		t.Fatalf("expected mode error, got %q", err.Error())
	}
}

// TestValidateRateLimiterRejectsInvalidWorkers ensures worker counts must be positive.
func TestValidateRateLimiterRejectsInvalidWorkers(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Workers = -1

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.workers") {
		t.Fatalf("expected workers error, got %q", err.Error())
	}
}

// TestValidateRateLimiterRejectsInvalidBatchSize ensures batch size must be positive.
func TestValidateRateLimiterRejectsInvalidBatchSize(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Batch.Size = -1

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.batch.size") {
		t.Fatalf("expected batch size error, got %q", err.Error())
	}
}

// TestValidateRateLimiterRejectsInvalidBatchFlush ensures batch flush intervals must be positive.
func TestValidateRateLimiterRejectsInvalidBatchFlush(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.Batch.FlushMs = -1

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.batch.flush_ms") {
		t.Fatalf("expected batch flush error, got %q", err.Error())
	}
}

// TestValidateRateLimiterRejectsInvalidTimeout ensures request timeout must be positive.
func TestValidateRateLimiterRejectsInvalidTimeout(t *testing.T) {
	cfg := validConfig()
	cfg.RateLimiter.RequestTimeoutMs = -1

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "rate_limiter.request_timeout_ms") {
		t.Fatalf("expected timeout error, got %q", err.Error())
	}
}

// TestValidateTaskConcurrencyRejectsNonPositive ensures invalid task concurrency is rejected.
func TestValidateTaskConcurrencyRejectsNonPositive(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks[0].Concurrency = -1

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "concurrency") {
		t.Fatalf("expected concurrency error, got %q", err.Error())
	}
}

// TestValidateTaskConcurrencyRequiresQuestionEval ensures only question_eval tasks accept concurrency.
func TestValidateTaskConcurrencyRequiresQuestionEval(t *testing.T) {
	cfg := validConfig()
	cfg.Tasks[0].Type = "other"
	cfg.Tasks[0].Concurrency = 2

	baseDir := t.TempDir()
	writeQuestionSpec(t, baseDir)
	err := validateWithTimeout(t, cfg, baseDir)
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if !strings.Contains(err.Error(), "concurrency") {
		t.Fatalf("expected concurrency error, got %q", err.Error())
	}
}

// validateWithTimeout runs validation with an explicit timeout.
func validateWithTimeout(t *testing.T, cfg spec.Config, baseDir string) error {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Validate(&cfg, baseDir)
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		t.Fatalf("validation timed out")
		return ctx.Err()
	}
}
