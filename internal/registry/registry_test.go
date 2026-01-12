package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
	"cogni/pkg/ratelimiter"
)

func TestRegistry_RoundTrip_SaveLoad(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		reg := New()
		reg.Put(sampleState("limit:a", 10, ratelimiter.LimitStatusActive, 0))
		reg.Put(sampleState("limit:b", 20, ratelimiter.LimitStatusDecreasing, 5))
		reg.Put(sampleState("limit:c", 30, ratelimiter.LimitStatusActive, 0))

		path := filepath.Join(t.TempDir(), "limits.json")
		if err := reg.Save(path); err != nil {
			t.Fatalf("save registry: %v", err)
		}

		loaded := New()
		if err := loaded.Load(path); err != nil {
			t.Fatalf("load registry: %v", err)
		}

		got := loaded.List()
		want := reg.List()
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected states: %#v", got)
		}
	})
}

func TestRegistry_AtomicWrite_NoTmpLeftBehind(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		reg := New()
		reg.Put(sampleState("limit:a", 10, ratelimiter.LimitStatusActive, 0))

		dir := t.TempDir()
		path := filepath.Join(dir, "limits.json")
		if err := reg.Save(path); err != nil {
			t.Fatalf("save registry: %v", err)
		}
		if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
			t.Fatalf("expected tmp file to be removed, got %v", err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read registry: %v", err)
		}
		var states []ratelimiter.LimitState
		if err := json.Unmarshal(data, &states); err != nil {
			t.Fatalf("parse registry json: %v", err)
		}
	})
}

func TestRegistry_ConcurrentAccess_NoRace(t *testing.T) {
	runWithTimeout(t, time.Second, func() {
		ctx := testutil.Context(t, 250*time.Millisecond)
		reg := New()
		state := sampleState("limit:race", 10, ratelimiter.LimitStatusActive, 0)
		reg.Put(state)

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
						_, _ = reg.Get(state.Definition.Key)
					}
				}
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					reg.Put(state)
				}
			}
		}()

		wg.Wait()
	})
}

func sampleState(key string, capacity uint64, status ratelimiter.LimitStatus, pending uint64) ratelimiter.LimitState {
	return ratelimiter.LimitState{
		Definition: ratelimiter.LimitDefinition{
			Key:            ratelimiter.LimitKey(key),
			Kind:           ratelimiter.KindRolling,
			Capacity:       capacity,
			WindowSeconds:  60,
			TimeoutSeconds: 0,
			Unit:           "requests",
			Description:    "test",
			Overage:        ratelimiter.OverageDebt,
		},
		Status:            status,
		PendingDecreaseTo: pending,
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
