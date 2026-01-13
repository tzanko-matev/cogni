package ratelimiter

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
)

type fakeBatchLimiter struct {
	mu            sync.Mutex
	reserveCalls  []BatchReserveRequest
	completeCalls []BatchCompleteRequest
	reserveHook   func(BatchReserveRequest) BatchReserveResponse
	completeHook  func(BatchCompleteRequest) BatchCompleteResponse
}

func (f *fakeBatchLimiter) Reserve(_ context.Context, _ ReserveRequest) (ReserveResponse, error) {
	return ReserveResponse{}, fmt.Errorf("reserve not used")
}

func (f *fakeBatchLimiter) Complete(_ context.Context, _ CompleteRequest) (CompleteResponse, error) {
	return CompleteResponse{}, fmt.Errorf("complete not used")
}

func (f *fakeBatchLimiter) BatchReserve(_ context.Context, req BatchReserveRequest) (BatchReserveResponse, error) {
	f.mu.Lock()
	f.reserveCalls = append(f.reserveCalls, req)
	hook := f.reserveHook
	f.mu.Unlock()
	if hook != nil {
		return hook(req), nil
	}
	results := make([]BatchReserveResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, BatchReserveResult{Allowed: true, ReservedAtUnix: time.Now().UnixMilli()})
	}
	return BatchReserveResponse{Results: results}, nil
}

func (f *fakeBatchLimiter) BatchComplete(_ context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error) {
	f.mu.Lock()
	f.completeCalls = append(f.completeCalls, req)
	hook := f.completeHook
	f.mu.Unlock()
	if hook != nil {
		return hook(req), nil
	}
	results := make([]BatchCompleteResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, BatchCompleteResult{Ok: true})
	}
	return BatchCompleteResponse{Results: results}, nil
}

func TestBatcher_FlushesWithinInterval(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeBatchLimiter{}
		called := make(chan struct{}, 1)
		lim.reserveHook = func(req BatchReserveRequest) BatchReserveResponse {
			select {
			case called <- struct{}{}:
			default:
			}
			results := make([]BatchReserveResult, 0, len(req.Requests))
			for range req.Requests {
				results = append(results, BatchReserveResult{Allowed: true, ReservedAtUnix: 123})
			}
			return BatchReserveResponse{Results: results}
		}

		batcher := NewBatcher(lim, 10, 20*time.Millisecond)
		defer func() {
			_ = batcher.Shutdown(testutil.Context(t, time.Second))
		}()

		errCh := make(chan error, 1)
		go func() {
			_, err := batcher.Reserve(context.Background(), ReserveRequest{
				LeaseID:      "L1",
				Requirements: []Requirement{{Key: "k1", Amount: 1}},
			})
			errCh <- err
		}()

		waitFor(t, called, 150*time.Millisecond)
		ctx := testutil.Context(t, 150*time.Millisecond)
		select {
		case <-ctx.Done():
			t.Fatalf("reserve did not return in time")
		case err := <-errCh:
			if err != nil {
				t.Fatalf("reserve error: %v", err)
			}
		}
	})
}

func TestBatcher_PreservesOrder(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeBatchLimiter{}
		mu := sync.Mutex{}
		expected := map[string]int64{}
		lim.reserveHook = func(req BatchReserveRequest) BatchReserveResponse {
			results := make([]BatchReserveResult, len(req.Requests))
			mu.Lock()
			for i, r := range req.Requests {
				reservedAt := int64(100 + i)
				expected[r.LeaseID] = reservedAt
				results[i] = BatchReserveResult{Allowed: true, ReservedAtUnix: reservedAt}
			}
			mu.Unlock()
			return BatchReserveResponse{Results: results}
		}
		batcher := NewBatcher(lim, 10, 20*time.Millisecond)
		defer func() {
			_ = batcher.Shutdown(testutil.Context(t, time.Second))
		}()

		type result struct {
			leaseID string
			resp    ReserveResponse
			err     error
		}
		results := make(chan result, 2)
		call := func(leaseID string) {
			resp, err := batcher.Reserve(context.Background(), ReserveRequest{
				LeaseID:      leaseID,
				Requirements: []Requirement{{Key: "k1", Amount: 1}},
			})
			results <- result{leaseID: leaseID, resp: resp, err: err}
		}

		go call("L1")
		go call("L2")

		ctx := testutil.Context(t, 200*time.Millisecond)
		for i := 0; i < 2; i++ {
			select {
			case <-ctx.Done():
				t.Fatalf("timeout waiting for responses")
			case res := <-results:
				if res.err != nil {
					t.Fatalf("reserve error: %v", res.err)
				}
				mu.Lock()
				want := expected[res.leaseID]
				mu.Unlock()
				if res.resp.ReservedAtUnixMs != want {
					t.Fatalf("lease %s expected reserved_at %d, got %d", res.leaseID, want, res.resp.ReservedAtUnixMs)
				}
			}
		}
	})
}

func TestBatcher_DoesNotMixReserveAndComplete(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		lim := &fakeBatchLimiter{}
		reserveCalled := make(chan struct{}, 1)
		completeCalled := make(chan struct{}, 1)
		lim.reserveHook = func(req BatchReserveRequest) BatchReserveResponse {
			select {
			case reserveCalled <- struct{}{}:
			default:
			}
			results := make([]BatchReserveResult, len(req.Requests))
			for i := range req.Requests {
				results[i] = BatchReserveResult{Allowed: true, ReservedAtUnix: int64(200 + i)}
			}
			return BatchReserveResponse{Results: results}
		}
		lim.completeHook = func(req BatchCompleteRequest) BatchCompleteResponse {
			select {
			case completeCalled <- struct{}{}:
			default:
			}
			results := make([]BatchCompleteResult, len(req.Requests))
			for i := range req.Requests {
				results[i] = BatchCompleteResult{Ok: true}
			}
			return BatchCompleteResponse{Results: results}
		}

		batcher := NewBatcher(lim, 10, 20*time.Millisecond)
		defer func() {
			_ = batcher.Shutdown(testutil.Context(t, time.Second))
		}()

		resErr := make(chan error, 1)
		compErr := make(chan error, 1)
		go func() {
			_, err := batcher.Reserve(context.Background(), ReserveRequest{
				LeaseID:      "R1",
				Requirements: []Requirement{{Key: "k1", Amount: 1}},
			})
			resErr <- err
		}()
		go func() {
			_, err := batcher.Complete(context.Background(), CompleteRequest{
				LeaseID: "R1",
				Actuals: []Actual{},
			})
			compErr <- err
		}()

		waitFor(t, reserveCalled, 200*time.Millisecond)
		waitFor(t, completeCalled, 200*time.Millisecond)

		ctx := testutil.Context(t, 200*time.Millisecond)
		for i := 0; i < 2; i++ {
			select {
			case <-ctx.Done():
				t.Fatalf("timeout waiting for batch results")
			case err := <-resErr:
				if err != nil {
					t.Fatalf("reserve error: %v", err)
				}
			case err := <-compErr:
				if err != nil {
					t.Fatalf("complete error: %v", err)
				}
			}
		}
	})
}
