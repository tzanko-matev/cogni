package memory

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

// BenchmarkMemoryReserve_1Key measures reserve throughput for a single rolling key.
func BenchmarkMemoryReserve_1Key(b *testing.B) {
	clock := testutil.NewFakeClock(time.Unix(0, 0))
	backend := New(clock)
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:rpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      uint64(b.N) + 100,
		WindowSeconds: 60,
		Unit:          "requests",
		Overage:       ratelimiter.OverageDebt,
	})
	reqs := []ratelimiter.Requirement{{Key: "bench:rpm", Amount: 1}}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaseID := strconv.Itoa(i)
		res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{
			LeaseID:      leaseID,
			Requirements: reqs,
		}, clock.Now())
		if err != nil || !res.Allowed {
			b.Fatalf("reserve failed: %v %+v", err, res)
		}
	}
}

// BenchmarkMemoryReserve_4Keys measures reserve throughput for four requirements.
func BenchmarkMemoryReserve_4Keys(b *testing.B) {
	clock := testutil.NewFakeClock(time.Unix(0, 0))
	backend := New(clock)
	capacity := uint64(b.N) + 100
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:rpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity,
		WindowSeconds: 60,
		Unit:          "requests",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:tpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity * 50,
		WindowSeconds: 60,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:daily",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity * 100,
		WindowSeconds: 3600,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:            "bench:concurrency",
		Kind:           ratelimiter.KindConcurrency,
		Capacity:       capacity,
		TimeoutSeconds: 300,
		Unit:           "requests",
		Overage:        ratelimiter.OverageDebt,
	})
	reqs := []ratelimiter.Requirement{
		{Key: "bench:rpm", Amount: 1},
		{Key: "bench:tpm", Amount: 50},
		{Key: "bench:daily", Amount: 50},
		{Key: "bench:concurrency", Amount: 1},
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaseID := strconv.Itoa(i)
		res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{
			LeaseID:      leaseID,
			Requirements: reqs,
		}, clock.Now())
		if err != nil || !res.Allowed {
			b.Fatalf("reserve failed: %v %+v", err, res)
		}
	}
}

// BenchmarkMemoryComplete_4Keys measures completion cost for multi-key leases.
func BenchmarkMemoryComplete_4Keys(b *testing.B) {
	clock := testutil.NewFakeClock(time.Unix(0, 0))
	backend := New(clock)
	capacity := uint64(b.N) + 100
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:rpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity,
		WindowSeconds: 60,
		Unit:          "requests",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:tpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity * 50,
		WindowSeconds: 60,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "bench:daily",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity * 100,
		WindowSeconds: 3600,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:            "bench:concurrency",
		Kind:           ratelimiter.KindConcurrency,
		Capacity:       capacity,
		TimeoutSeconds: 300,
		Unit:           "requests",
		Overage:        ratelimiter.OverageDebt,
	})
	reqs := []ratelimiter.Requirement{
		{Key: "bench:rpm", Amount: 1},
		{Key: "bench:tpm", Amount: 50},
		{Key: "bench:daily", Amount: 50},
		{Key: "bench:concurrency", Amount: 1},
	}
	actuals := []ratelimiter.Actual{
		{Key: "bench:tpm", ActualAmount: 40},
		{Key: "bench:daily", ActualAmount: 40},
	}
	ctx := context.Background()
	leases := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		leaseID := strconv.Itoa(i)
		res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{
			LeaseID:      leaseID,
			Requirements: reqs,
		}, clock.Now())
		if err != nil || !res.Allowed {
			b.Fatalf("reserve failed: %v %+v", err, res)
		}
		leases[i] = leaseID
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := backend.Complete(ctx, ratelimiter.CompleteRequest{
			LeaseID: leases[i],
			Actuals: actuals,
		})
		if err != nil {
			b.Fatalf("complete failed: %v", err)
		}
	}
}

// BenchmarkScheduler_Throughput_MemoryBackend measures scheduler throughput with memory backend.
func BenchmarkScheduler_Throughput_MemoryBackend(b *testing.B) {
	clock := testutil.NewFakeClock(time.Unix(0, 0))
	backend := New(clock)
	capacity := uint64(b.N) + 100
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "global:llm:bench:unit:rpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity,
		WindowSeconds: 60,
		Unit:          "requests",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:           "global:llm:bench:unit:tpm",
		Kind:          ratelimiter.KindRolling,
		Capacity:      capacity * 50,
		WindowSeconds: 60,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	})
	applyBenchDef(b, backend, ratelimiter.LimitDefinition{
		Key:            "global:llm:bench:unit:concurrency",
		Kind:           ratelimiter.KindConcurrency,
		Capacity:       capacity,
		TimeoutSeconds: 300,
		Unit:           "requests",
		Overage:        ratelimiter.OverageDebt,
	})

	limiter := benchLimiter{backend: backend, now: clock.Now}
	workers := runtime.GOMAXPROCS(0)
	if workers < 2 {
		workers = 2
	}
	scheduler := ratelimiter.NewScheduler(limiter, workers)
	defer func() {
		_ = scheduler.Shutdown(context.Background())
	}()

	var wg sync.WaitGroup
	wg.Add(b.N)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		leaseID := strconv.Itoa(i)
		jobID := "job-" + leaseID
		scheduler.Submit(ratelimiter.Job{
			JobID:           jobID,
			LeaseID:         leaseID,
			TenantID:        "bench-tenant",
			Provider:        "bench",
			Model:           "unit",
			Prompt:          "hello",
			MaxOutputTokens: 10,
			Execute: func(context.Context) (uint64, error) {
				wg.Done()
				return 10, nil
			},
		})
	}
	wg.Wait()
}

// benchLimiter adapts MemoryBackend to the Limiter interface for benchmarks.
type benchLimiter struct {
	backend *MemoryBackend
	now     func() time.Time
}

// Reserve forwards reserve requests to the memory backend.
func (b benchLimiter) Reserve(ctx context.Context, req ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	return b.backend.Reserve(ctx, req, b.now())
}

// Complete forwards completion requests to the memory backend.
func (b benchLimiter) Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	return b.backend.Complete(ctx, req)
}

// BatchReserve executes batch reserve requests sequentially for benchmarks.
func (b benchLimiter) BatchReserve(ctx context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
	for _, item := range req.Requests {
		res, err := b.backend.Reserve(ctx, item, b.now())
		if err != nil {
			results = append(results, ratelimiter.BatchReserveResult{Allowed: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchReserveResult{
			Allowed:        res.Allowed,
			RetryAfterMs:   res.RetryAfterMs,
			ReservedAtUnix: res.ReservedAtUnixMs,
			Error:          res.Error,
		})
	}
	return ratelimiter.BatchReserveResponse{Results: results}, nil
}

// BatchComplete executes batch complete requests sequentially for benchmarks.
func (b benchLimiter) BatchComplete(ctx context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
	for _, item := range req.Requests {
		res, err := b.backend.Complete(ctx, item)
		if err != nil {
			results = append(results, ratelimiter.BatchCompleteResult{Ok: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchCompleteResult{Ok: res.Ok, Error: res.Error})
	}
	return ratelimiter.BatchCompleteResponse{Results: results}, nil
}

// applyBenchDef applies a limit definition for benchmark setup.
func applyBenchDef(b *testing.B, backend *MemoryBackend, def ratelimiter.LimitDefinition) {
	b.Helper()
	if err := backend.ApplyDefinition(context.Background(), def); err != nil {
		b.Fatalf("apply definition: %v", err)
	}
}
