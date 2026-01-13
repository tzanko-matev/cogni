//go:build chaos && integration

package chaos

import (
	"context"
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

// TestChaos_TBRestart_ServerRecovers verifies the server recovers after TB restarts.
func TestChaos_TBRestart_ServerRecovers(t *testing.T) {
	runWithTimeout(t, 25*time.Second, func() {
		instance := testutil.StartTigerBeetleSingleReplica(t)
		server, backend := startTBServer(t, instance)
		defer server.Close()
		defer backend.Close()

		defs := basicDefs()
		for _, def := range defs {
			ratelimitertest.HTTPPutLimit(t, server.BaseURL, def)
		}

		stop := make(chan struct{})
		var wg sync.WaitGroup
		var errCount uint64
		client := httpclient.New(server.BaseURL)
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
					}
					leaseID := ratelimiter.NewULID()
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					res, err := client.Reserve(ctx, reserveReq(leaseID, 10))
					cancel()
					if err != nil {
						atomic.AddUint64(&errCount, 1)
						continue
					}
					if !res.Allowed {
						continue
					}
					ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
					_, err = client.Complete(ctx, completeReq(leaseID, 10))
					cancel()
					if err != nil {
						atomic.AddUint64(&errCount, 1)
					}
				}
			}()
		}

		time.Sleep(2 * time.Second)
		instance.Stop()
		time.Sleep(2 * time.Second)

		close(stop)
		wg.Wait()

		newInstance := testutil.StartTigerBeetleSingleReplica(t)
		newServer, newBackend := startTBServer(t, newInstance)
		defer newServer.Close()
		defer newBackend.Close()
		for _, def := range defs {
			ratelimitertest.HTTPPutLimit(t, newServer.BaseURL, def)
		}

		testutil.Eventually(t, 5*time.Second, 100*time.Millisecond, func() bool {
			leaseID := ratelimiter.NewULID()
			res := ratelimitertest.HTTPReserve(t, newServer.BaseURL, reserveReq(leaseID, 10))
			return res.Allowed
		}, "expected server to recover after TB restart")
	})
}

// TestChaos_ServerRestart_InFlightReservationsExpire ensures expiries unblock after restart.
func TestChaos_ServerRestart_InFlightReservationsExpire(t *testing.T) {
	runWithTimeout(t, 20*time.Second, func() {
		instance := testutil.StartTigerBeetleSingleReplica(t)
		server, backend := startTBServer(t, instance)
		defs := basicDefs()
		defs[2].TimeoutSeconds = 2
		defs[0].WindowSeconds = 2
		defs[1].WindowSeconds = 2
		for _, def := range defs {
			ratelimitertest.HTTPPutLimit(t, server.BaseURL, def)
		}

		res1 := ratelimitertest.HTTPReserve(t, server.BaseURL, reserveReq("lease-a", 10))
		if !res1.Allowed {
			t.Fatalf("expected allow")
		}
		res2 := ratelimitertest.HTTPReserve(t, server.BaseURL, reserveReq("lease-b", 10))
		if !res2.Allowed {
			t.Fatalf("expected allow")
		}

		server.Close()
		_ = backend.Close()

		newServer, newBackend := startTBServer(t, instance)
		defer newServer.Close()
		defer newBackend.Close()
		for _, def := range defs {
			ratelimitertest.HTTPPutLimit(t, newServer.BaseURL, def)
		}

		time.Sleep(3 * time.Second)
		testutil.Eventually(t, 4*time.Second, 100*time.Millisecond, func() bool {
			leaseID := ratelimiter.NewULID()
			res := ratelimitertest.HTTPReserve(t, newServer.BaseURL, reserveReq(leaseID, 10))
			return res.Allowed
		}, "expected reservations to allow after restart")
	})
}

// startTBServer wires a TB backend with a test HTTP server.
func startTBServer(t *testing.T, instance *testutil.TBInstance) (*ratelimitertest.ServerInstance, *tb.Backend) {
	t.Helper()
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
	server := ratelimitertest.StartServer(t, ratelimitertest.ServerConfig{
		Registry: reg,
		Backend:  backend,
	})
	return server, backend
}

// basicDefs returns a baseline set of rolling and concurrency limits.
func basicDefs() []ratelimiter.LimitDefinition {
	return []ratelimiter.LimitDefinition{
		{
			Key:           "global:llm:openai:model:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      100,
			WindowSeconds: 3,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           "global:llm:openai:model:tpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      2000,
			WindowSeconds: 3,
			Unit:          "tokens",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:            "global:llm:openai:model:concurrency",
			Kind:           ratelimiter.KindConcurrency,
			Capacity:       10,
			TimeoutSeconds: 3,
			Unit:           "requests",
			Overage:        ratelimiter.OverageDebt,
		},
	}
}

// reserveReq constructs a reserve request for the chaos suite.
func reserveReq(leaseID string, tpm uint64) ratelimiter.ReserveRequest {
	return ratelimiter.ReserveRequest{
		LeaseID: leaseID,
		Requirements: []ratelimiter.Requirement{
			{Key: "global:llm:openai:model:rpm", Amount: 1},
			{Key: "global:llm:openai:model:tpm", Amount: tpm},
			{Key: "global:llm:openai:model:concurrency", Amount: 1},
		},
	}
}

// completeReq constructs a completion request for the chaos suite.
func completeReq(leaseID string, tpm uint64) ratelimiter.CompleteRequest {
	return ratelimiter.CompleteRequest{
		LeaseID: leaseID,
		Actuals: []ratelimiter.Actual{{Key: "global:llm:openai:model:tpm", ActualAmount: tpm}},
	}
}

// runWithTimeout enforces a hard timeout for chaos tests.
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
