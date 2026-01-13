package bench

import (
	"context"
	"strconv"
	"testing"
	"time"

	"cogni/internal/backend/memory"
	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
	"cogni/pkg/ratelimiter/httpclient"
)

// BenchmarkHTTPReserve_1Key measures reserve latency over HTTP for a single key.
func BenchmarkHTTPReserve_1Key(b *testing.B) {
	server := startHTTPBenchServer(b, []ratelimiter.LimitDefinition{
		{
			Key:           "bench:http:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      uint64(b.N) + 100,
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
	})
	client := httpclient.New(server.BaseURL)
	req := ratelimiter.ReserveRequest{
		Requirements: []ratelimiter.Requirement{{Key: "bench:http:rpm", Amount: 1}},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.LeaseID = strconv.Itoa(i)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.Reserve(ctx, req)
		cancel()
		if err != nil {
			b.Fatalf("reserve error: %v", err)
		}
	}
}

// BenchmarkHTTPReserve_4Keys measures reserve latency over HTTP for four keys.
func BenchmarkHTTPReserve_4Keys(b *testing.B) {
	capacity := uint64(b.N) + 100
	server := startHTTPBenchServer(b, []ratelimiter.LimitDefinition{
		{
			Key:           "bench:http:rpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      capacity,
			WindowSeconds: 60,
			Unit:          "requests",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           "bench:http:tpm",
			Kind:          ratelimiter.KindRolling,
			Capacity:      capacity * 50,
			WindowSeconds: 60,
			Unit:          "tokens",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:           "bench:http:daily",
			Kind:          ratelimiter.KindRolling,
			Capacity:      capacity * 100,
			WindowSeconds: 3600,
			Unit:          "tokens",
			Overage:       ratelimiter.OverageDebt,
		},
		{
			Key:            "bench:http:concurrency",
			Kind:           ratelimiter.KindConcurrency,
			Capacity:       capacity,
			TimeoutSeconds: 300,
			Unit:           "requests",
			Overage:        ratelimiter.OverageDebt,
		},
	})
	client := httpclient.New(server.BaseURL)
	req := ratelimiter.ReserveRequest{
		Requirements: []ratelimiter.Requirement{
			{Key: "bench:http:rpm", Amount: 1},
			{Key: "bench:http:tpm", Amount: 50},
			{Key: "bench:http:daily", Amount: 50},
			{Key: "bench:http:concurrency", Amount: 1},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.LeaseID = strconv.Itoa(i)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, err := client.Reserve(ctx, req)
		cancel()
		if err != nil {
			b.Fatalf("reserve error: %v", err)
		}
	}
}

// startHTTPBenchServer starts an in-memory ratelimiterd server for benchmarks.
func startHTTPBenchServer(b *testing.B, defs []ratelimiter.LimitDefinition) *testutil.ServerInstance {
	b.Helper()
	reg := registry.New()
	backend := memory.New(nil)
	server := testutil.StartServer(b, testutil.ServerConfig{
		Registry: reg,
		Backend:  backend,
	})
	b.Cleanup(server.Close)
	for _, def := range defs {
		testutil.HTTPPutLimit(b, server.BaseURL, def)
	}
	return server
}
