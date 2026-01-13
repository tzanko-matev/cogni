//go:build stress

package stress

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cogni/internal/backend/memory"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

// TestStress_Memory_RandomizedWorkload exercises randomized concurrent traffic in memory backend.
func TestStress_Memory_RandomizedWorkload(t *testing.T) {
	runWithTimeout(t, 20*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := memory.New(clock)
		ctx := context.Background()

		providers := []string{"openai", "anthropic", "google"}
		keys := make([]providerKeys, 0, len(providers))
		for _, provider := range providers {
			keys = append(keys, providerKeys{
				rpm:  ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:model:rpm", provider)),
				tpm:  ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:model:tpm", provider)),
				conc: ratelimiter.LimitKey(fmt.Sprintf("global:llm:%s:model:concurrency", provider)),
			})
		}

		for _, key := range keys {
			applyDef(t, backend, ratelimiter.LimitDefinition{
				Key:           key.rpm,
				Kind:          ratelimiter.KindRolling,
				Capacity:      50,
				WindowSeconds: 2,
				Unit:          "requests",
				Overage:       ratelimiter.OverageDebt,
			})
			applyDef(t, backend, ratelimiter.LimitDefinition{
				Key:           key.tpm,
				Kind:          ratelimiter.KindRolling,
				Capacity:      2000,
				WindowSeconds: 2,
				Unit:          "tokens",
				Overage:       ratelimiter.OverageDebt,
			})
			applyDef(t, backend, ratelimiter.LimitDefinition{
				Key:            key.conc,
				Kind:           ratelimiter.KindConcurrency,
				Capacity:       10,
				TimeoutSeconds: 2,
				Unit:           "requests",
				Overage:        ratelimiter.OverageDebt,
			})
		}

		stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		var wg sync.WaitGroup
		var allowedCount uint64

		workerCount := 100
		for i := 0; i < workerCount; i++ {
			wg.Add(1)
			go func(seed int64) {
				defer wg.Done()
				rng := rand.New(rand.NewSource(seed))
				counter := 0
				for {
					select {
					case <-stopCtx.Done():
						return
					default:
					}
					idx := rng.Intn(len(keys))
					selected := keys[idx]
					counter++
					leaseID := fmt.Sprintf("stress-%d-%d", seed, counter)
					upper := uint64(rng.Intn(200) + 1)
					reqs := []ratelimiter.Requirement{
						{Key: selected.rpm, Amount: 1},
						{Key: selected.tpm, Amount: upper},
						{Key: selected.conc, Amount: 1},
					}
					res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{
						LeaseID:      leaseID,
						Requirements: reqs,
					}, clock.Now())
					if err != nil || !res.Allowed {
						continue
					}
					atomic.AddUint64(&allowedCount, 1)
					actual := uint64(rng.Intn(int(upper)) + 1)
					clock.Advance(time.Duration(rng.Intn(50)+1) * time.Millisecond)
					_, _ = backend.Complete(ctx, ratelimiter.CompleteRequest{
						LeaseID: leaseID,
						Actuals: []ratelimiter.Actual{{Key: selected.tpm, ActualAmount: actual}},
					})
					time.Sleep(time.Duration(rng.Intn(5)+1) * time.Millisecond)
				}
			}(int64(i + 1))
		}

		wg.Wait()
		if atomic.LoadUint64(&allowedCount) == 0 {
			t.Fatalf("expected some allowed reservations")
		}
		snapshot := backend.DebugSnapshot()
		for key, roll := range snapshot.Rolling {
			if roll.Used > roll.Capacity {
				t.Fatalf("rolling %s exceeds capacity %d > %d", key, roll.Used, roll.Capacity)
			}
		}
		for key, conc := range snapshot.Concurrency {
			if uint64(conc.Holds) > conc.Capacity {
				t.Fatalf("concurrency %s exceeds capacity %d > %d", key, conc.Holds, conc.Capacity)
			}
		}
	})
}

// providerKeys groups the per-provider limit keys for stress generation.
type providerKeys struct {
	rpm  ratelimiter.LimitKey
	tpm  ratelimiter.LimitKey
	conc ratelimiter.LimitKey
}

// applyDef applies a limit definition with a short timeout.
func applyDef(t *testing.T, backend *memory.MemoryBackend, def ratelimiter.LimitDefinition) {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	if err := backend.ApplyDefinition(ctx, def); err != nil {
		t.Fatalf("apply definition: %v", err)
	}
}

// runWithTimeout enforces a hard timeout for stress tests.
func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	ctx := testutil.Context(t, timeout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-ctx.Done():
		t.Fatalf("test timed out")
	case <-done:
	}
}
