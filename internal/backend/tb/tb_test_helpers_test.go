//go:build integration

package tb

import (
	"strconv"
	"testing"
	"time"

	"cogni/internal/registry"
	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func newTBBackendForTest(t *testing.T) *Backend {
	t.Helper()
	instance := testutil.StartTigerBeetleSingleReplica(t)
	clusterID, err := strconv.ParseUint(instance.ClusterID, 10, 32)
	if err != nil {
		t.Fatalf("parse cluster id: %v", err)
	}
	reg := registry.New()
	backend, err := New(Config{
		ClusterID:      uint32(clusterID),
		Addresses:      instance.Addresses,
		Sessions:       1,
		MaxBatchEvents: 8000,
		FlushInterval:  200 * time.Microsecond,
		Registry:       reg,
	})
	if err != nil {
		t.Fatalf("new backend: %v", err)
	}
	t.Cleanup(func() {
		_ = backend.Close()
	})
	return backend
}

func applyDefinition(t *testing.T, backend *Backend, def ratelimiter.LimitDefinition) {
	t.Helper()
	ctx := testutil.Context(t, 4*time.Second)
	if err := backend.ApplyDefinition(ctx, def); err != nil {
		t.Fatalf("apply definition: %v", err)
	}
}

func reserve(t *testing.T, backend *Backend, leaseID string, reqs []ratelimiter.Requirement) ratelimiter.ReserveResponse {
	t.Helper()
	ctx := testutil.Context(t, 4*time.Second)
	res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{LeaseID: leaseID, Requirements: reqs}, time.Now())
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	return res
}

func complete(t *testing.T, backend *Backend, leaseID string, actuals []ratelimiter.Actual) {
	t.Helper()
	ctx := testutil.Context(t, 4*time.Second)
	res, err := backend.Complete(ctx, ratelimiter.CompleteRequest{LeaseID: leaseID, Actuals: actuals})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if !res.Ok {
		t.Fatalf("complete returned ok=false")
	}
}

func req(key string, amount uint64) []ratelimiter.Requirement {
	return []ratelimiter.Requirement{{Key: ratelimiter.LimitKey(key), Amount: amount}}
}

func multiReq(reqs ...[]ratelimiter.Requirement) []ratelimiter.Requirement {
	var out []ratelimiter.Requirement
	for _, r := range reqs {
		out = append(out, r...)
	}
	return out
}

func rollingDef(key string, cap uint64, window int) ratelimiter.LimitDefinition {
	return ratelimiter.LimitDefinition{
		Key:           ratelimiter.LimitKey(key),
		Kind:          ratelimiter.KindRolling,
		Capacity:      cap,
		WindowSeconds: window,
		Unit:          "tokens",
		Overage:       ratelimiter.OverageDebt,
	}
}

func concDef(key string, cap uint64, timeout int) ratelimiter.LimitDefinition {
	return ratelimiter.LimitDefinition{
		Key:            ratelimiter.LimitKey(key),
		Kind:           ratelimiter.KindConcurrency,
		Capacity:       cap,
		TimeoutSeconds: timeout,
		Unit:           "requests",
		Overage:        ratelimiter.OverageDebt,
	}
}

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

func runWithBackend(t *testing.T, timeout time.Duration, fn func(*Backend)) {
	t.Helper()
	backend := newTBBackendForTest(t)
	runWithTimeout(t, timeout, func() {
		fn(backend)
	})
}
