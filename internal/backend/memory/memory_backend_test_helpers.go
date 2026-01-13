package memory

import (
	"fmt"
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func newMemoryBackendForTest(clock *testutil.FakeClock) *MemoryBackend {
	return New(clock)
}

func applyDefs(t *testing.T, backend *MemoryBackend, defs ...ratelimiter.LimitDefinition) {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	for _, def := range defs {
		if err := backend.ApplyDefinition(ctx, def); err != nil {
			t.Fatalf("apply definition: %v", err)
		}
	}
}

func reserve(t *testing.T, backend *MemoryBackend, leaseID string, reqs []ratelimiter.Requirement, now time.Time) ratelimiter.ReserveResponse {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	res, err := backend.Reserve(ctx, ratelimiter.ReserveRequest{LeaseID: leaseID, Requirements: reqs}, now)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	return res
}

func allowReserve(t *testing.T, backend *MemoryBackend, leaseID string, reqs []ratelimiter.Requirement, now time.Time) {
	t.Helper()
	res := reserve(t, backend, leaseID, reqs, now)
	if !res.Allowed {
		t.Fatalf("expected allow, got %+v", res)
	}
}

func complete(t *testing.T, backend *MemoryBackend, leaseID string, actuals []ratelimiter.Actual) {
	t.Helper()
	ctx := testutil.Context(t, time.Second)
	_, err := backend.Complete(ctx, ratelimiter.CompleteRequest{LeaseID: leaseID, Actuals: actuals})
	if err != nil {
		t.Fatalf("complete: %v", err)
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

// newRand returns a lightweight pseudo-random generator.
func newRand(seed int64) *lockedRand {
	return &lockedRand{state: seed}
}

// lockedRand is a simple linear congruential generator for tests.
type lockedRand struct {
	state int64
}

func (r *lockedRand) Intn(n int) int {
	return int(r.next() % int64(n))
}

func (r *lockedRand) Int() int {
	return int(r.next())
}

func (r *lockedRand) next() int64 {
	const a = int64(48271)
	const m = int64(2147483647)
	r.state = (a * r.state) % m
	return r.state
}

func leaseID(seed int64, next int) string {
	return fmt.Sprintf("L-%d-%d", seed, next)
}
