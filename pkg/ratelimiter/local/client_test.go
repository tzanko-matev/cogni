package local

import (
	"context"
	"testing"
	"time"

	"cogni/pkg/ratelimiter"
)

// TestNewMemoryLimiterFromStatesAllowsReserve verifies the in-memory limiter accepts limits.
func TestNewMemoryLimiterFromStatesAllowsReserve(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		limiter, err := NewMemoryLimiterFromStates([]ratelimiter.LimitState{
			{
				Definition: ratelimiter.LimitDefinition{
					Key:            "global:llm:test:model:concurrency",
					Kind:           ratelimiter.KindConcurrency,
					Capacity:       1,
					TimeoutSeconds: 1,
					Unit:           "requests",
					Overage:        ratelimiter.OverageDebt,
				},
				Status: ratelimiter.LimitStatusActive,
			},
		})
		if err != nil {
			t.Fatalf("create limiter: %v", err)
		}
		req := ratelimiter.ReserveRequest{
			LeaseID: "lease-1",
			Requirements: []ratelimiter.Requirement{{
				Key:    "global:llm:test:model:concurrency",
				Amount: 1,
			}},
		}
		res, err := limiter.Reserve(context.Background(), req)
		if err != nil {
			t.Fatalf("reserve: %v", err)
		}
		if !res.Allowed {
			t.Fatalf("expected allowed reservation, got %+v", res)
		}
	})
}

func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		t.Fatalf("test timed out")
	}
}
