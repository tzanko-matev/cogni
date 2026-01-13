package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrBatcherClosed is returned when the batcher has been shut down.
var ErrBatcherClosed = errors.New("batcher is closed")

// batchKind identifies the operation type stored in a batch item.
type batchKind uint8

const (
	// batchKindReserve marks reserve requests.
	batchKindReserve batchKind = iota
	// batchKindComplete marks complete requests.
	batchKindComplete
)

// batchItem captures a single Reserve or Complete request awaiting batching.
type batchItem struct {
	kind        batchKind
	reserve     ReserveRequest
	complete    CompleteRequest
	reserveCh   chan ReserveResponse
	completeCh  chan CompleteResponse
	errCh       chan error
	deadline    time.Time
	hasDeadline bool
}

// Batcher aggregates Reserve and Complete calls into batch requests.
type Batcher struct {
	limiter       Limiter
	maxBatch      int
	flushInterval time.Duration

	in       chan batchItem
	stopCh   chan struct{}
	doneCh   chan struct{}
	stopOnce sync.Once
}

// NewBatcher constructs a Batcher for the provided limiter.
func NewBatcher(limiter Limiter, maxBatch int, flushInterval time.Duration) *Batcher {
	if maxBatch <= 0 {
		maxBatch = 1
	}
	if flushInterval <= 0 {
		flushInterval = 2 * time.Millisecond
	}
	b := &Batcher{
		limiter:       limiter,
		maxBatch:      maxBatch,
		flushInterval: flushInterval,
		in:            make(chan batchItem, maxBatch*2),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}
	go b.run()
	return b
}

// Reserve queues a reserve request and waits for the batched response.
func (b *Batcher) Reserve(ctx context.Context, req ReserveRequest) (ReserveResponse, error) {
	item := batchItem{
		kind:      batchKindReserve,
		reserve:   req,
		reserveCh: make(chan ReserveResponse, 1),
		errCh:     make(chan error, 1),
	}
	if deadline, ok := ctx.Deadline(); ok {
		item.deadline = deadline
		item.hasDeadline = true
	}
	if err := b.enqueue(ctx, item); err != nil {
		return ReserveResponse{}, err
	}
	select {
	case res := <-item.reserveCh:
		return res, nil
	case err := <-item.errCh:
		return ReserveResponse{}, err
	case <-ctx.Done():
		return ReserveResponse{}, ctx.Err()
	case <-b.doneCh:
		return ReserveResponse{}, ErrBatcherClosed
	}
}

// Complete queues a complete request and waits for the batched response.
func (b *Batcher) Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error) {
	item := batchItem{
		kind:       batchKindComplete,
		complete:   req,
		completeCh: make(chan CompleteResponse, 1),
		errCh:      make(chan error, 1),
	}
	if deadline, ok := ctx.Deadline(); ok {
		item.deadline = deadline
		item.hasDeadline = true
	}
	if err := b.enqueue(ctx, item); err != nil {
		return CompleteResponse{}, err
	}
	select {
	case res := <-item.completeCh:
		return res, nil
	case err := <-item.errCh:
		return CompleteResponse{}, err
	case <-ctx.Done():
		return CompleteResponse{}, ctx.Err()
	case <-b.doneCh:
		return CompleteResponse{}, ErrBatcherClosed
	}
}

// BatchReserve forwards batch reserve requests to the underlying limiter.
func (b *Batcher) BatchReserve(ctx context.Context, req BatchReserveRequest) (BatchReserveResponse, error) {
	return b.limiter.BatchReserve(ctx, req)
}

// BatchComplete forwards batch complete requests to the underlying limiter.
func (b *Batcher) BatchComplete(ctx context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error) {
	return b.limiter.BatchComplete(ctx, req)
}

// Shutdown stops the batcher loop after flushing queued work.
func (b *Batcher) Shutdown(ctx context.Context) error {
	b.stopOnce.Do(func() { close(b.stopCh) })
	select {
	case <-b.doneCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// enqueue submits a batch item for processing or returns an error.
func (b *Batcher) enqueue(ctx context.Context, item batchItem) error {
	select {
	case <-b.doneCh:
		return ErrBatcherClosed
	case <-ctx.Done():
		return ctx.Err()
	case b.in <- item:
		return nil
	}
}
