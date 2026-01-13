//go:build stress && integration

package stress

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cogni/internal/backend/tb"
	"cogni/internal/ratelimitertest"
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
)

// TestStress_TB_RandomizedWorkload runs randomized load against the TB-backed server.
func TestStress_TB_RandomizedWorkload(t *testing.T) {
	runWithTimeout(t, 30*time.Second, func() {
		instance := testutil.StartTigerBeetleSingleReplica(t)
		clusterID, err := strconv.ParseUint(instance.ClusterID, 10, 32)
		if err != nil {
			t.Fatalf("parse cluster id: %v", err)
		}
		reg := registry.New()
		backend, err := tb.New(tb.Config{
			ClusterID:      uint32(clusterID),
			Addresses:      instance.Addresses,
			Sessions:       1,
			MaxBatchEvents: 8000,
			FlushInterval:  200 * time.Microsecond,
			Registry:       reg,
		})
		if err != nil {
			t.Fatalf("tb backend: %v", err)
		}
		defer func() {
			_ = backend.Close()
		}()
		server := ratelimitertest.StartServer(t, ratelimitertest.ServerConfig{
			Registry: reg,
			Backend:  backend,
		})
		defer server.Close()

		defs := []ratelimiter.LimitDefinition{
			{
				Key:           "global:llm:openai:model:rpm",
				Kind:          ratelimiter.KindRolling,
				Capacity:      100,
				WindowSeconds: 2,
				Unit:          "requests",
				Overage:       ratelimiter.OverageDebt,
			},
			{
				Key:           "global:llm:openai:model:tpm",
				Kind:          ratelimiter.KindRolling,
				Capacity:      2000,
				WindowSeconds: 2,
				Unit:          "tokens",
				Overage:       ratelimiter.OverageDebt,
			},
			{
				Key:            "global:llm:openai:model:concurrency",
				Kind:           ratelimiter.KindConcurrency,
				Capacity:       10,
				TimeoutSeconds: 15,
				Unit:           "requests",
				Overage:        ratelimiter.OverageDebt,
			},
		}
		for _, def := range defs {
			ratelimitertest.HTTPPutLimit(t, server.BaseURL, def)
		}

		lim := httpclient.New(server.BaseURL)
		stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		var wg sync.WaitGroup
		var allowedCount uint64
		var errorCount uint64
		var maxPending uint64

		pollCtx, pollCancel := context.WithCancel(context.Background())
		defer pollCancel()
		go func() {
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-pollCtx.Done():
					return
				case <-ticker.C:
				}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				pending, err := backend.DebugPendingDebits(ctx, "global:llm:openai:model:concurrency")
				cancel()
				if err != nil {
					continue
				}
				updateMaxUint64(&maxPending, pending)
			}
		}()

		workerCount := 32
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(seed int64) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(seed))
				for {
					select {
					case <-stopCtx.Done():
						return
					default:
					}
					leaseID := ratelimiter.NewULID()
					upper := uint64(rng.Intn(200) + 1)
					req := ratelimiter.ReserveRequest{
						LeaseID: leaseID,
						Requirements: []ratelimiter.Requirement{
							{Key: "global:llm:openai:model:rpm", Amount: 1},
							{Key: "global:llm:openai:model:tpm", Amount: upper},
							{Key: "global:llm:openai:model:concurrency", Amount: 1},
						},
					}
					ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
					res, err := lim.Reserve(ctx, req)
					cancel()
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
						time.Sleep(time.Duration(rng.Intn(5)+1) * time.Millisecond)
						continue
					}
					if !res.Allowed {
						time.Sleep(time.Duration(rng.Intn(5)+1) * time.Millisecond)
						continue
					}
					atomic.AddUint64(&allowedCount, 1)
					time.Sleep(time.Duration(rng.Intn(50)) * time.Millisecond)
					actual := uint64(rng.Intn(int(upper)) + 1)
					ctx, cancel = context.WithTimeout(context.Background(), 4*time.Second)
					_, err = lim.Complete(ctx, ratelimiter.CompleteRequest{
						LeaseID: leaseID,
						Actuals: []ratelimiter.Actual{{Key: "global:llm:openai:model:tpm", ActualAmount: actual}},
					})
					cancel()
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
					}
				}
			}(int64(i + 1))
		}

		wg.Wait()
		pollCancel()
		if atomic.LoadUint64(&errorCount) != 0 {
			t.Fatalf("expected zero HTTP errors, got %d", errorCount)
		}
		if atomic.LoadUint64(&allowedCount) == 0 {
			t.Fatalf("expected some allowed reservations")
		}
		if atomic.LoadUint64(&maxPending) > 10 {
			t.Fatalf("max pending %d exceeds cap", maxPending)
		}
	})
}

// updateMaxUint64 tracks the maximum observed uint64 value.
func updateMaxUint64(max *uint64, value uint64) {
	for {
		current := atomic.LoadUint64(max)
		if value <= current {
			return
		}
		if atomic.CompareAndSwapUint64(max, current, value) {
			return
		}
	}
}
