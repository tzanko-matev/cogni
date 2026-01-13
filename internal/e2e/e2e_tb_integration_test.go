//go:build integration

package e2e

import (
	"context"
	"strconv"
	"testing"
	"time"

	"cogni/internal/backend/tb"
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
)

// TestE2E_TB_AdminDefineThenReserve verifies admin-defined limits allow reservations immediately.
func TestE2E_TB_AdminDefineThenReserve(t *testing.T) {
	runWithTimeout(t, 10*time.Second, func() {
		server, backend := startTBServer(t)
		defer server.Close()
		defer backend.Close()

		def := ratelimiter.LimitDefinition{
			Key:           "global:llm:test:model:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      2,
			WindowSeconds: 3,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		}
		testutil.HTTPPutLimit(t, server.BaseURL, def)
		res := testutil.HTTPReserve(t, server.BaseURL, ratelimiter.ReserveRequest{
			LeaseID:      "e2e-lease-1",
			Requirements: []ratelimiter.Requirement{{Key: def.Key, Amount: 1}},
		})
		if !res.Allowed {
			t.Fatalf("expected allow, got %+v", res)
		}
	})
}

// TestE2E_TB_ReserveComplete_Idempotency asserts Reserve/Complete idempotency over HTTP.
func TestE2E_TB_ReserveComplete_Idempotency(t *testing.T) {
	runWithTimeout(t, 12*time.Second, func() {
		server, backend := startTBServer(t)
		defer server.Close()
		defer backend.Close()

		def := ratelimiter.LimitDefinition{
			Key:           "global:llm:test:model:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      1,
			WindowSeconds: 3,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		}
		testutil.HTTPPutLimit(t, server.BaseURL, def)
		req := ratelimiter.ReserveRequest{
			LeaseID:      "e2e-lease-2",
			Requirements: []ratelimiter.Requirement{{Key: def.Key, Amount: 1}},
		}
		first := testutil.HTTPReserve(t, server.BaseURL, req)
		if !first.Allowed {
			t.Fatalf("expected allow on first reserve, got %+v", first)
		}
		second := testutil.HTTPReserve(t, server.BaseURL, req)
		if !second.Allowed {
			t.Fatalf("expected idempotent allow, got %+v", second)
		}
		completeReq := ratelimiter.CompleteRequest{LeaseID: req.LeaseID}
		res := testutil.HTTPComplete(t, server.BaseURL, completeReq)
		if !res.Ok {
			t.Fatalf("expected complete ok, got %+v", res)
		}
		res = testutil.HTTPComplete(t, server.BaseURL, completeReq)
		if !res.Ok {
			t.Fatalf("expected idempotent complete ok, got %+v", res)
		}
	})
}

// TestE2E_TB_Scheduler_NoHOL_WithRealServer ensures denied queues do not block others.
func TestE2E_TB_Scheduler_NoHOL_WithRealServer(t *testing.T) {
	runWithTimeout(t, 15*time.Second, func() {
		server, backend := startTBServer(t)
		defer server.Close()
		defer backend.Close()

		applyLLMLimits(t, server.BaseURL, "openai", "gpt-4o-mini", 0, 0, 1)
		applyLLMLimits(t, server.BaseURL, "anthropic", "claude-3-haiku", 5, 500, 2)

		client := httpclient.New(server.BaseURL)
		scheduler := ratelimiter.NewScheduler(client, 2)
		defer func() {
			_ = scheduler.Shutdown(testutil.Context(t, 2*time.Second))
		}()

		done := make(chan string, 2)
		makeJob := func(provider, model, jobID string) ratelimiter.Job {
			return ratelimiter.Job{
				JobID:           jobID,
				TenantID:        "tenant-1",
				Provider:        provider,
				Model:           model,
				Prompt:          "hello",
				MaxOutputTokens: 10,
				Execute: func(ctx context.Context) (uint64, error) {
					done <- jobID
					return 5, nil
				},
			}
		}

		scheduler.Submit(makeJob("openai", "gpt-4o-mini", "openai-1"))
		scheduler.Submit(makeJob("anthropic", "claude-3-haiku", "anthropic-1"))
		scheduler.Submit(makeJob("anthropic", "claude-3-haiku", "anthropic-2"))

		ctx := testutil.Context(t, 2*time.Second)
		received := map[string]struct{}{}
		for len(received) < 2 {
			select {
			case <-ctx.Done():
				t.Fatalf("expected anthropic jobs to complete without HOL blocking")
			case jobID := <-done:
				received[jobID] = struct{}{}
			}
		}
	})
}

// TestE2E_TB_BatchReserveAndComplete validates batch ordering and idempotency.
func TestE2E_TB_BatchReserveAndComplete(t *testing.T) {
	runWithTimeout(t, 12*time.Second, func() {
		server, backend := startTBServer(t)
		defer server.Close()
		defer backend.Close()

		def := ratelimiter.LimitDefinition{
			Key:           "global:llm:test:model:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      1,
			WindowSeconds: 3,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		}
		testutil.HTTPPutLimit(t, server.BaseURL, def)

		client := httpclient.New(server.BaseURL)
		ctx := testutil.Context(t, 2*time.Second)
		batchRes, err := client.BatchReserve(ctx, ratelimiter.BatchReserveRequest{
			Requests: []ratelimiter.ReserveRequest{
				{LeaseID: "batch-1", Requirements: []ratelimiter.Requirement{{Key: def.Key, Amount: 1}}},
				{LeaseID: "batch-2", Requirements: []ratelimiter.Requirement{{Key: def.Key, Amount: 1}}},
			},
		})
		if err != nil {
			t.Fatalf("batch reserve error: %v", err)
		}
		if len(batchRes.Results) != 2 {
			t.Fatalf("expected 2 batch results, got %d", len(batchRes.Results))
		}
		if !batchRes.Results[0].Allowed || batchRes.Results[1].Allowed {
			t.Fatalf("expected ordered allow/deny, got %+v", batchRes.Results)
		}

		ctx = testutil.Context(t, 2*time.Second)
		again, err := client.BatchReserve(ctx, ratelimiter.BatchReserveRequest{
			Requests: []ratelimiter.ReserveRequest{
				{LeaseID: "batch-1", Requirements: []ratelimiter.Requirement{{Key: def.Key, Amount: 1}}},
			},
		})
		if err != nil {
			t.Fatalf("idempotent batch reserve error: %v", err)
		}
		if len(again.Results) != 1 || !again.Results[0].Allowed {
			t.Fatalf("expected idempotent allow, got %+v", again.Results)
		}

		ctx = testutil.Context(t, 2*time.Second)
		completeRes, err := client.BatchComplete(ctx, ratelimiter.BatchCompleteRequest{
			Requests: []ratelimiter.CompleteRequest{{LeaseID: "batch-1"}},
		})
		if err != nil {
			t.Fatalf("batch complete error: %v", err)
		}
		if len(completeRes.Results) != 1 || !completeRes.Results[0].Ok {
			t.Fatalf("expected batch complete ok, got %+v", completeRes)
		}
		ctx = testutil.Context(t, 2*time.Second)
		completeRes, err = client.BatchComplete(ctx, ratelimiter.BatchCompleteRequest{
			Requests: []ratelimiter.CompleteRequest{{LeaseID: "batch-1"}},
		})
		if err != nil {
			t.Fatalf("idempotent batch complete error: %v", err)
		}
		if len(completeRes.Results) != 1 || !completeRes.Results[0].Ok {
			t.Fatalf("expected idempotent batch complete ok, got %+v", completeRes)
		}
	})
}

// startTBServer launches a TB backend and HTTP server for integration tests.
func startTBServer(t *testing.T) (*testutil.ServerInstance, *tb.Backend) {
	t.Helper()
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
	server := testutil.StartServer(t, testutil.ServerConfig{
		Registry: reg,
		Backend:  backend,
	})
	return server, backend
}

// applyLLMLimits writes a standard rpm/tpm/concurrency trio for a provider/model pair.
func applyLLMLimits(t *testing.T, baseURL, provider, model string, rpmCap, tpmCap, concCap uint64) {
	t.Helper()
	defs := []ratelimiter.LimitDefinition{
		{
			Key:           ratelimiter.LimitKey("global:llm:" + provider + ":" + model + ":rpm"),
			Kind:          ratelimiter.KindRolling,
			Capacity:      rpmCap,
			WindowSeconds: 2,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           ratelimiter.LimitKey("global:llm:" + provider + ":" + model + ":tpm"),
			Kind:          ratelimiter.KindRolling,
			Capacity:      tpmCap,
			WindowSeconds: 2,
			Unit:          "tokens",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:            ratelimiter.LimitKey("global:llm:" + provider + ":" + model + ":concurrency"),
			Kind:           ratelimiter.KindConcurrency,
			Capacity:       concCap,
			TimeoutSeconds: 2,
			Unit:           "requests",
			Overage:        ratelimiter.OverageDebt,
		},
	}
	for _, def := range defs {
		testutil.HTTPPutLimit(t, baseURL, def)
	}
}

// runWithTimeout enforces a hard timeout for e2e integration tests.
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
