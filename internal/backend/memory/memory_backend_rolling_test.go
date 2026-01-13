package memory

import (
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestMemory_Rolling_AllowThenDeny(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 2, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		allowReserve(t, backend, "L2", req("k1", 1), clock.Now())
		res := reserve(t, backend, "L3", req("k1", 1), clock.Now())
		if res.Allowed {
			t.Fatalf("expected deny")
		}
		if res.RetryAfterMs <= 0 {
			t.Fatalf("expected retry_after_ms > 0")
		}
	})
}

func TestMemory_Rolling_ExpiryReleasesCapacity(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 1, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		clock.Advance(11 * time.Second)
		allowReserve(t, backend, "L2", req("k1", 1), clock.Now())
	})
}

func TestMemory_MultiKeyAtomicity_NoPartialReserve(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 1, 10), rollingDef("k2", 0, 10))

		res := reserve(t, backend, "L1", multiReq(req("k1", 1), req("k2", 1)), clock.Now())
		if res.Allowed {
			t.Fatalf("expected deny")
		}
		allowReserve(t, backend, "L2", req("k1", 1), clock.Now())
	})
}

func TestMemory_ReconcileFreesSlack(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 100, 10))

		allowReserve(t, backend, "L1", req("k1", 100), clock.Now())
		complete(t, backend, "L1", []ratelimiter.Actual{{Key: "k1", ActualAmount: 10}})
		allowReserve(t, backend, "L2", req("k1", 90), clock.Now())
	})
}

func TestMemory_OverageRecordsDebt(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		def := rollingDef("k1", 100, 10)
		def.Overage = ratelimiter.OverageDebt
		applyDefs(t, backend, def)

		allowReserve(t, backend, "D1", req("k1", 100), clock.Now())
		complete(t, backend, "D1", []ratelimiter.Actual{{Key: "k1", ActualAmount: 140}})
		if backend.DebtForKey("k1") != 40 {
			t.Fatalf("expected debt 40, got %d", backend.DebtForKey("k1"))
		}
	})
}

func TestMemory_ReserveIdempotent_NoDoubleCount(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 1, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		res := reserve(t, backend, "L2", req("k1", 1), clock.Now())
		if res.Allowed {
			t.Fatalf("expected deny")
		}
	})
}

func TestMemory_CompleteIdempotent_NoError(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 1, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		complete(t, backend, "L1", nil)
		complete(t, backend, "L1", nil)
		complete(t, backend, "missing", nil)
	})
}
