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
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
)

// TestStress_TB_RandomizedWorkload runs randomized load against the TB-backed server.
func TestStress_TB_RandomizedWorkload(t *testing.T) {
	runWithTimeout(t, 20*time.Second, func() {
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
		server := testutil.StartServer(t, testutil.ServerConfig{
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
				TimeoutSeconds: 2,
				Unit:           "requests",
				Overage:        ratelimiter.OverageDebt,
			},
		}
		for _, def := range defs {
			testutil.HTTPPutLimit(t, server.BaseURL, def)
		}

		lim := httpclient.New(server.BaseURL)
		stop := time.After(10 * time.Second)
		var wg sync.WaitGroup
		var allowedCount uint64
		var errorCount uint64
		var inFlight int64
		var maxInFlight int64

		for i := 0; i < 200; i++ {
			wg.Add(1)
			go func(seed int64) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(seed))
				counter := 0
				for {
					select {
					case <-stop:
						return
					default:
					}
					counter++
					leaseID := testutil.NewULID()
					upper := uint64(rng.Intn(200) + 1)
					req := ratelimiter.ReserveRequest{
						LeaseID: leaseID,
						Requirements: []ratelimiter.Requirement{
							{Key: "global:llm:openai:model:rpm", Amount: 1},
							{Key: "global:llm:openai:model:tpm", Amount: upper},
							{Key: "global:llm:openai:model:concurrency", Amount: 1},
						},
					}
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					res, err := lim.Reserve(ctx, req)
					cancel()
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
						continue
					}
					if !res.Allowed {
						continue
					}
					atomic.AddUint64(&allowedCount, 1)
					current := atomic.AddInt64(&inFlight, 1)
					updateMax(&maxInFlight, current)
					time.Sleep(time.Duration(rng.Intn(50)) * time.Millisecond)
					actual := uint64(rng.Intn(int(upper)) + 1)
					ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
					_, err = lim.Complete(ctx, ratelimiter.CompleteRequest{
						LeaseID: leaseID,
						Actuals: []ratelimiter.Actual{{Key: "global:llm:openai:model:tpm", ActualAmount: actual}},
					})
					cancel()
					atomic.AddInt64(&inFlight, -1)
					if err != nil {
						atomic.AddUint64(&errorCount, 1)
					}
				}
			}(int64(i + 1))
		}

		wg.Wait()
		if atomic.LoadUint64(&errorCount) != 0 {
			t.Fatalf("expected zero HTTP errors, got %d", errorCount)
		}
		if atomic.LoadUint64(&allowedCount) == 0 {
			t.Fatalf("expected some allowed reservations")
		}
		if atomic.LoadInt64(&maxInFlight) > 10 {
			t.Fatalf("max in-flight %d exceeds cap", maxInFlight)
		}
	})
}

// updateMax tracks the maximum observed in-flight count.
func updateMax(max *int64, value int64) {
	for {
		current := atomic.LoadInt64(max)
		if value <= current {
			return
		}
		if atomic.CompareAndSwapInt64(max, current, value) {
			return
		}
	}
}
