//go:build integration

package tb

import (
	"fmt"
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestTB_ApplyDefinition_CanReserveImmediately(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("rpm", 10, 2))
		applyDefinition(t, backend, rollingDef("tpm", 100, 2))
		applyDefinition(t, backend, concDef("conc", 2, 5))

		reqs := multiReq(req("rpm", 1), req("tpm", 10), req("conc", 1))
		res := reserve(t, backend, "lease-1", reqs)
		if !res.Allowed {
			t.Fatalf("expected allow")
		}
	})
}

func TestTB_MultiKeyAtomicity_LinkedAllOrNothing(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 1, 3))
		applyDefinition(t, backend, rollingDef("k2", 0, 3))

		res := reserve(t, backend, "lease-1", multiReq(req("k1", 1), req("k2", 1)))
		if res.Allowed {
			t.Fatalf("expected deny")
		}
		res2 := reserve(t, backend, "lease-2", req("k1", 1))
		if !res2.Allowed {
			t.Fatalf("expected allow for k1")
		}
	})
}

func TestTB_DeniedAttemptMustUseNewLeaseID(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 1, 2))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected initial allow")
		}
		if reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected deny")
		}

		time.Sleep(3 * time.Second)
		if reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected deny on same lease")
		}

		testutil.Eventually(t, 3*time.Second, 50*time.Millisecond, func() bool {
			leaseID := ratelimiter.NewULID()
			return reserve(t, backend, leaseID, req("k1", 1)).Allowed
		}, "expected allow with new lease")
	})
}

func TestTB_ReserveIdempotent_AllowedDoesNotDoubleCount(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 1, 3))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected idempotent allow")
		}
		if reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected deny")
		}
	})
}

func TestTB_Concurrency_ReleasedOnComplete(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, concDef("k1", 1, 10))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		if reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected deny")
		}
		complete(t, backend, "lease-a", nil)
		if !reserve(t, backend, "lease-c", req("k1", 1)).Allowed {
			t.Fatalf("expected allow after complete")
		}
	})
}

func TestTB_Concurrency_TimeoutReleases(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, concDef("k1", 1, 2))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		time.Sleep(3 * time.Second)
		if !reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected allow after timeout")
		}
	})
}

func TestTB_ReconcileFreesSlack(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 100, 3))

		if !reserve(t, backend, "lease-a", req("k1", 100)).Allowed {
			t.Fatalf("expected allow")
		}
		complete(t, backend, "lease-a", []ratelimiter.Actual{{Key: "k1", ActualAmount: 10}})

		testutil.Eventually(t, 2*time.Second, 50*time.Millisecond, func() bool {
			leaseID := ratelimiter.NewULID()
			return reserve(t, backend, leaseID, req("k1", 90)).Allowed
		}, "expected allow after reconcile")
	})
}

func TestTB_OverageRecordsDebt(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		def := rollingDef("k1", 100, 3)
		def.Overage = ratelimiter.OverageDebt
		applyDefinition(t, backend, def)

		if !reserve(t, backend, "lease-a", req("k1", 100)).Allowed {
			t.Fatalf("expected allow")
		}
		complete(t, backend, "lease-a", []ratelimiter.Actual{{Key: "k1", ActualAmount: 140}})

		ctx := testutil.Context(t, 2*time.Second)
		debt, err := backend.DebtForKey(ctx, ratelimiter.LimitKey("k1"))
		if err != nil {
			t.Fatalf("debt lookup: %v", err)
		}
		if debt != 40 {
			t.Fatalf("expected debt 40, got %d", debt)
		}
	})
}

func TestTB_DynamicLimitCreation_NoRestartRequired(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 5, 2))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow after definition")
		}
	})
}

func TestTB_CapacityIncrease_TakesEffectImmediately(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 1, 2))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		if reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected deny")
		}
		applyDefinition(t, backend, rollingDef("k1", 2, 2))
		if !reserve(t, backend, "lease-c", req("k1", 1)).Allowed {
			t.Fatalf("expected allow after increase")
		}
	})
}

func TestTB_CapacityDecrease_BlocksUntilApplied(t *testing.T) {
	runWithBackend(t, 10*time.Second, func(backend *Backend) {
		applyDefinition(t, backend, rollingDef("k1", 2, 2))

		if !reserve(t, backend, "lease-a", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		if !reserve(t, backend, "lease-b", req("k1", 1)).Allowed {
			t.Fatalf("expected allow")
		}
		applyDefinition(t, backend, rollingDef("k1", 1, 2))
		res := reserve(t, backend, "lease-c", req("k1", 1))
		if res.Allowed || res.Error != "limit_decreasing:k1" {
			t.Fatalf("expected limit_decreasing, got %+v", res)
		}

		var last ratelimiter.ReserveResponse
		testutil.Eventually(t, 6*time.Second, 100*time.Millisecond, func() bool {
			leaseID := ratelimiter.NewULID()
			last = reserve(t, backend, leaseID, req("k1", 1))
			return last.Allowed
		}, fmt.Sprintf("expected allow after decrease applied, last=%+v", last))
	})
}
