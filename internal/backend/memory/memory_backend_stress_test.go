package memory

import (
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestMemory_ConcurrentStress_NoRacesAndNeverExceeds(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend,
			rollingDef("rpm", 50, 2),
			rollingDef("tpm", 2000, 2),
			concDef("conc", 10, 2),
		)

		ctx := testutil.Context(t, 500*time.Millisecond)
		errCh := make(chan error, 1)
		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			seed := int64(i + 1)
			wg.Add(1)
			go func(seed int64) {
				defer wg.Done()
				rng := newRand(seed)
				for {
					select {
					case <-ctx.Done():
						return
					default:
						lease := leaseID(seed, rng.Int())
						reqs := multiReq(req("rpm", 1), req("tpm", uint64(rng.Intn(200)+1)), req("conc", 1))
						res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{LeaseID: lease, Requirements: reqs}, clock.Now())
						if err != nil {
							select {
							case errCh <- err:
							default:
							}
							return
						}
						if res.Allowed {
							clock.Advance(time.Duration(rng.Intn(5)) * time.Millisecond)
							if _, err := backend.Complete(ctx, ratelimiter.CompleteRequest{LeaseID: lease, Actuals: []ratelimiter.Actual{{Key: "tpm", ActualAmount: reqs[1].Amount}}}); err != nil {
								select {
								case errCh <- err:
								default:
								}
								return
							}
						}
					}
				}
			}(seed)
		}

		<-ctx.Done()
		wg.Wait()
		select {
		case err := <-errCh:
			t.Fatalf("stress error: %v", err)
		default:
		}

		for key, limit := range backend.roll {
			if limit.used > limit.cap {
				t.Fatalf("rolling limit %s exceeded: used=%d cap=%d", key, limit.used, limit.cap)
			}
		}
		for key, limit := range backend.conc {
			if uint64(len(limit.holds)) > limit.cap {
				t.Fatalf("concurrency limit %s exceeded: holds=%d cap=%d", key, len(limit.holds), limit.cap)
			}
		}
	})
}
