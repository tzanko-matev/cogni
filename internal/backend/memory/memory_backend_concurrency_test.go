package memory

import (
	"testing"
	"time"

	"cogni/internal/testutil"
)

func TestMemory_Concurrency_ReleaseOnComplete(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, concDef("k1", 1, 300))

		allowReserve(t, backend, "C1", req("k1", 1), clock.Now())
		res := reserve(t, backend, "C2", req("k1", 1), clock.Now())
		if res.Allowed {
			t.Fatalf("expected deny")
		}
		complete(t, backend, "C1", nil)
		allowReserve(t, backend, "C3", req("k1", 1), clock.Now())
	})
}

func TestMemory_Concurrency_TimeoutReleases(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		clock := testutil.NewFakeClock(time.Unix(0, 0))
		backend := newMemoryBackendForTest(clock)
		applyDefs(t, backend, concDef("k1", 1, 3))

		allowReserve(t, backend, "C1", req("k1", 1), clock.Now())
		clock.Advance(4 * time.Second)
		allowReserve(t, backend, "C2", req("k1", 1), clock.Now())
	})
}
