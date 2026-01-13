package tbutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"cogni/internal/testutil"
	tb "github.com/tigerbeetledb/tigerbeetle-go"
	tbtypes "github.com/tigerbeetledb/tigerbeetle-go/pkg/types"
)

// fakeClient captures submitter batch sizes without hitting a real TigerBeetle server.
type fakeClient struct {
	mu      sync.Mutex
	batches []int
	callCh  chan struct{}
}

func (f *fakeClient) CreateAccounts(_ []tbtypes.Account) ([]tbtypes.AccountEventResult, error) {
	return nil, nil
}

func (f *fakeClient) CreateTransfers(transfers []tbtypes.Transfer) ([]tbtypes.TransferEventResult, error) {
	f.mu.Lock()
	f.batches = append(f.batches, len(transfers))
	f.mu.Unlock()
	if f.callCh != nil {
		select {
		case f.callCh <- struct{}{}:
		default:
		}
	}
	return nil, nil
}

func (f *fakeClient) LookupAccounts(_ []tbtypes.Uint128) ([]tbtypes.Account, error) {
	return nil, nil
}

func (f *fakeClient) LookupTransfers(_ []tbtypes.Uint128) ([]tbtypes.Transfer, error) {
	return nil, nil
}

func (f *fakeClient) GetAccountTransfers(_ tbtypes.AccountFilter) ([]tbtypes.Transfer, error) {
	return nil, nil
}

func (f *fakeClient) GetAccountBalances(_ tbtypes.AccountFilter) ([]tbtypes.AccountBalance, error) {
	return nil, nil
}

func (f *fakeClient) QueryAccounts(_ tbtypes.QueryFilter) ([]tbtypes.Account, error) {
	return nil, nil
}

func (f *fakeClient) QueryTransfers(_ tbtypes.QueryFilter) ([]tbtypes.Transfer, error) {
	return nil, nil
}

func (f *fakeClient) Nop() error {
	return nil
}

func (f *fakeClient) Close() error {
	return nil
}

// newSubmitterForTest builds a Submitter with an in-memory client pool.
func newSubmitterForTest(t *testing.T, maxEvents int, flush time.Duration, client *fakeClient) (*Submitter, context.CancelFunc) {
	t.Helper()
	if client == nil {
		client = &fakeClient{}
	}
	pool := &ClientPool{
		clients:   []tb.Client{client},
		available: make(chan tb.Client, 1),
	}
	pool.available <- client
	submitter := &Submitter{
		In:         make(chan WorkItem, 128),
		FlushEvery: flush,
		MaxEvents:  maxEvents,
		Pool:       pool,
	}
	ctx, cancel := context.WithCancel(context.Background())
	go submitter.Run(ctx)
	return submitter, cancel
}

// TestSubmitter_DoesNotSplitWorkItem verifies oversized work items fail immediately.
func TestSubmitter_DoesNotSplitWorkItem(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		submitter, cancel := newSubmitterForTest(t, 5, 5*time.Millisecond, nil)
		defer cancel()

		item := WorkItem{
			Transfers: make([]tbtypes.Transfer, 6),
			Done:      make(chan WorkResult, 1),
		}
		submitter.In <- item
		ctx := testutil.Context(t, 200*time.Millisecond)
		select {
		case <-ctx.Done():
			t.Fatalf("work item did not complete")
		case result := <-item.Done:
			if result.Err == nil {
				t.Fatalf("expected error for oversized item")
			}
		}
	})
}

// TestSubmitter_FlushesWithinInterval ensures the flush ticker drains items promptly.
func TestSubmitter_FlushesWithinInterval(t *testing.T) {
	runWithTimeout(t, 2*time.Second, func() {
		client := &fakeClient{callCh: make(chan struct{}, 1)}
		submitter, cancel := newSubmitterForTest(t, 10, 5*time.Millisecond, client)
		defer cancel()

		item := WorkItem{
			Transfers: []tbtypes.Transfer{{}},
			Done:      make(chan WorkResult, 1),
		}
		submitter.In <- item

		ctx := testutil.Context(t, 50*time.Millisecond)
		select {
		case <-ctx.Done():
			t.Fatalf("submitter did not flush in time")
		case <-item.Done:
		}
	})
}

// TestSubmitter_BatchSizeNeverExceedsMax asserts each batch respects MaxEvents.
func TestSubmitter_BatchSizeNeverExceedsMax(t *testing.T) {
	runWithTimeout(t, 3*time.Second, func() {
		client := &fakeClient{}
		submitter, cancel := newSubmitterForTest(t, 5, 2*time.Millisecond, client)
		defer cancel()

		items := make([]WorkItem, 100)
		for i := range items {
			items[i] = WorkItem{
				Transfers: []tbtypes.Transfer{{}},
				Done:      make(chan WorkResult, 1),
			}
			submitter.In <- items[i]
		}

		ctx := testutil.Context(t, 500*time.Millisecond)
		for i := range items {
			select {
			case <-ctx.Done():
				t.Fatalf("timed out waiting for work items")
			case <-items[i].Done:
			}
		}

		client.mu.Lock()
		batches := append([]int(nil), client.batches...)
		client.mu.Unlock()
		if len(batches) == 0 {
			t.Fatalf("no batches recorded")
		}
		for _, size := range batches {
			if size > 5 {
				t.Fatalf("batch size %d exceeds max", size)
			}
		}
	})
}

// runWithTimeout enforces a hard timeout for longer-running submitter tests.
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
