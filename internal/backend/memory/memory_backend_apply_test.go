package memory

import (
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestMemory_ApplyDefinition_IncreaseCapacityTakesEffectImmediately(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 1, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		res := reserve(t, backend, "L2", req("k1", 1), clock.Now())
		if res.Allowed {
			t.Fatalf("expected deny")
		}

		applyDefs(t, backend, rollingDef("k1", 2, 10))
		allowReserve(t, backend, "L3", req("k1", 1), clock.Now())
	})
}

func TestMemory_ApplyDefinition_DecreaseCapacity_BlocksUntilApplied(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, rollingDef("k1", 2, 10))

		allowReserve(t, backend, "L1", req("k1", 1), clock.Now())
		allowReserve(t, backend, "L2", req("k1", 1), clock.Now())

		applyDefs(t, backend, rollingDef("k1", 1, 10))
		state := backend.states["k1"]
		if state.Status != ratelimiter.LimitStatusDecreasing {
			t.Fatalf("expected decreasing state, got %q", state.Status)
		}
		res := reserve(t, backend, "L3", req("k1", 1), clock.Now())
		if res.Allowed || res.Error != "limit_decreasing:k1" {
			t.Fatalf("expected limit_decreasing error, got %+v", res)
		}

		clock.Advance(11 * time.Second)
		backend.TryApplyDecrease("k1")
		allowReserve(t, backend, "L4", req("k1", 1), clock.Now())
	})
}
