package ratelimiter

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
)

type fakeLimiter struct {
	mu            sync.Mutex
	reserveCalls  []ReserveRequest
	completeCalls []CompleteRequest
	reserveFn     func(ReserveRequest) (ReserveResponse, error)
	completeCh    chan struct{}
}

func (f *fakeLimiter) Reserve(_ context.Context, req ReserveRequest) (ReserveResponse, error) {
	f.mu.Lock()
	f.reserveCalls = append(f.reserveCalls, req)
	fn := f.reserveFn
	f.mu.Unlock()
	if fn != nil {
		return fn(req)
	}
	return ReserveResponse{Allowed: true}, nil
}

func (f *fakeLimiter) Complete(_ context.Context, req CompleteRequest) (CompleteResponse, error) {
	f.mu.Lock()
	f.completeCalls = append(f.completeCalls, req)
	completeCh := f.completeCh
	f.mu.Unlock()
	if completeCh != nil {
		select {
		case completeCh <- struct{}{}:
		default:
		}
	}
	return CompleteResponse{Ok: true}, nil
}

func (f *fakeLimiter) BatchReserve(_ context.Context, _ BatchReserveRequest) (BatchReserveResponse, error) {
	return BatchReserveResponse{}, fmt.Errorf("batch reserve not used")
}

func (f *fakeLimiter) BatchComplete(_ context.Context, _ BatchCompleteRequest) (BatchCompleteResponse, error) {
	return BatchCompleteResponse{}, fmt.Errorf("batch complete not used")
}

type idSource struct {
	mu   sync.Mutex
	next int
}

func (s *idSource) Next() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	return fmt.Sprintf("L-%d", s.next)
}

func TestScheduler_NoHeadOfLineBlocking(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeLimiter{}
		lim.reserveFn = func(req ReserveRequest) (ReserveResponse, error) {
			if hasProvider(req.Requirements, "openai") {
				return ReserveResponse{Allowed: false, RetryAfterMs: 100}, nil
			}
			return ReserveResponse{Allowed: true}, nil
		}
		ids := &idSource{}
		cfg := schedulerConfig{
			now:             time.Now,
			newLeaseID:      ids.Next,
			jitter:          func(time.Duration) time.Duration { return 0 },
			errorRetryDelay: 5 * time.Millisecond,
			idleInterval:    time.Millisecond,
		}
		sched := newScheduler(lim, 1, cfg)
		defer func() {
			_ = sched.Shutdown(testutil.Context(t, time.Second))
		}()

		done := make(chan struct{}, 2)
		sched.Submit(Job{
			Provider: "openai",
			Model:    "gpt",
			Execute: func(context.Context) (uint64, error) {
				t.Fatalf("openai job should not execute")
				return 0, nil
			},
		})
		sched.Submit(Job{
			Provider: "anthropic",
			Model:    "claude",
			Execute: func(context.Context) (uint64, error) {
				done <- struct{}{}
				return 1, nil
			},
		})
		sched.Submit(Job{
			Provider: "anthropic",
			Model:    "claude",
			Execute: func(context.Context) (uint64, error) {
				done <- struct{}{}
				return 1, nil
			},
		})

		waitForCount(t, done, 2, 200*time.Millisecond)
	})
}

func TestScheduler_RetryUsesNewLeaseID(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeLimiter{}
		countMu := sync.Mutex{}
		callCounts := map[string]int{}
		lim.reserveFn = func(req ReserveRequest) (ReserveResponse, error) {
			countMu.Lock()
			count := callCounts[req.JobID]
			callCounts[req.JobID] = count + 1
			countMu.Unlock()
			if count == 0 {
				return ReserveResponse{Allowed: false, RetryAfterMs: 1}, nil
			}
			return ReserveResponse{Allowed: true}, nil
		}
		ids := &idSource{}
		cfg := schedulerConfig{
			now:             time.Now,
			newLeaseID:      ids.Next,
			jitter:          func(time.Duration) time.Duration { return 0 },
			errorRetryDelay: 5 * time.Millisecond,
			idleInterval:    time.Millisecond,
		}
		sched := newScheduler(lim, 1, cfg)
		defer func() {
			_ = sched.Shutdown(testutil.Context(t, time.Second))
		}()

		done := make(chan struct{})
		sched.Submit(Job{
			JobID:    "job-1",
			Provider: "openai",
			Model:    "gpt",
			Execute: func(context.Context) (uint64, error) {
				close(done)
				return 1, nil
			},
		})
		waitFor(t, done, 300*time.Millisecond)

		lim.mu.Lock()
		calls := append([]ReserveRequest(nil), lim.reserveCalls...)
		lim.mu.Unlock()
		if len(calls) < 2 {
			t.Fatalf("expected at least 2 reserve calls, got %d", len(calls))
		}
		if calls[0].LeaseID == calls[1].LeaseID {
			t.Fatalf("expected different lease IDs for retry")
		}
	})
}

func TestScheduler_CompleteAlwaysCalledAfterAllowed(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeLimiter{}
		completeCalled := make(chan struct{}, 2)
		lim.completeCh = completeCalled
		ids := &idSource{}
		cfg := schedulerConfig{
			now:             time.Now,
			newLeaseID:      ids.Next,
			jitter:          func(time.Duration) time.Duration { return 0 },
			errorRetryDelay: 5 * time.Millisecond,
			idleInterval:    time.Millisecond,
		}
		sched := newScheduler(lim, 2, cfg)
		defer func() {
			_ = sched.Shutdown(testutil.Context(t, time.Second))
		}()

		for i := 0; i < 2; i++ {
			sched.Submit(Job{
				JobID:    fmt.Sprintf("job-%d", i),
				Provider: "anthropic",
				Model:    "claude",
				Execute: func(context.Context) (uint64, error) {
					return 42, fmt.Errorf("execute error")
				},
			})
		}
		waitForCount(t, completeCalled, 2, 200*time.Millisecond)

		lim.mu.Lock()
		count := len(lim.completeCalls)
		lim.mu.Unlock()
		if count != 2 {
			t.Fatalf("expected 2 complete calls, got %d", count)
		}
	})
}

func hasProvider(reqs []Requirement, provider string) bool {
	for _, req := range reqs {
		if strings.Contains(string(req.Key), ":"+provider+":") {
			return true
		}
	}
	return false
}
