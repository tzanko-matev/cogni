package ratelimiter

import (
	"context"
	"errors"
	"time"
)

// run drives the batcher loop until shutdown.
func (b *Batcher) run() {
	timer := time.NewTimer(b.flushInterval)
	defer timer.Stop()

	var pending []batchItem
	flush := func() {
		if len(pending) == 0 {
			return
		}
		items := pending
		pending = nil
		b.flush(items)
	}

	for {
		select {
		case <-b.stopCh:
			flush()
			close(b.doneCh)
			return
		case item := <-b.in:
			pending = append(pending, item)
			if len(pending) >= b.maxBatch {
				flush()
				resetTimer(timer, b.flushInterval)
			}
		case <-timer.C:
			flush()
			resetTimer(timer, b.flushInterval)
		}
	}
}

// flush dispatches batched items by kind without mixing Reserve and Complete.
func (b *Batcher) flush(items []batchItem) {
	var reserveItems []batchItem
	var completeItems []batchItem
	for _, item := range items {
		switch item.kind {
		case batchKindReserve:
			reserveItems = append(reserveItems, item)
		case batchKindComplete:
			completeItems = append(completeItems, item)
		}
	}
	if len(reserveItems) > 0 {
		b.flushReserve(reserveItems)
	}
	if len(completeItems) > 0 {
		b.flushComplete(completeItems)
	}
}

// flushReserve submits a batch reserve request and routes responses.
func (b *Batcher) flushReserve(items []batchItem) {
	reqs := make([]ReserveRequest, 0, len(items))
	for _, item := range items {
		reqs = append(reqs, item.reserve)
	}
	ctx, cancel := batchContext(items)
	defer cancel()

	resp, err := b.limiter.BatchReserve(ctx, BatchReserveRequest{Requests: reqs})
	if err != nil || len(resp.Results) != len(items) {
		sendReserveError(items, err)
		return
	}
	for i, item := range items {
		result := resp.Results[i]
		item.reserveCh <- ReserveResponse{
			Allowed:          result.Allowed,
			RetryAfterMs:     result.RetryAfterMs,
			ReservedAtUnixMs: result.ReservedAtUnix,
			Error:            result.Error,
		}
	}
}

// flushComplete submits a batch complete request and routes responses.
func (b *Batcher) flushComplete(items []batchItem) {
	reqs := make([]CompleteRequest, 0, len(items))
	for _, item := range items {
		reqs = append(reqs, item.complete)
	}
	ctx, cancel := batchContext(items)
	defer cancel()

	resp, err := b.limiter.BatchComplete(ctx, BatchCompleteRequest{Requests: reqs})
	if err != nil || len(resp.Results) != len(items) {
		sendCompleteError(items, err)
		return
	}
	for i, item := range items {
		result := resp.Results[i]
		item.completeCh <- CompleteResponse{Ok: result.Ok, Error: result.Error}
	}
}

// batchContext derives a shared context for a batch from item deadlines.
func batchContext(items []batchItem) (context.Context, context.CancelFunc) {
	earliest, ok := earliestDeadline(items)
	if !ok {
		return context.Background(), func() {}
	}
	ctx, cancel := context.WithDeadline(context.Background(), earliest)
	return ctx, cancel
}

// earliestDeadline returns the earliest deadline across all items.
func earliestDeadline(items []batchItem) (time.Time, bool) {
	var earliest time.Time
	ok := false
	for _, item := range items {
		if !item.hasDeadline {
			continue
		}
		if !ok || item.deadline.Before(earliest) {
			earliest = item.deadline
			ok = true
		}
	}
	return earliest, ok
}

// sendReserveError reports a shared error to reserve callers.
func sendReserveError(items []batchItem, err error) {
	if err == nil {
		err = errors.New("batch reserve response mismatch")
	}
	for _, item := range items {
		item.errCh <- err
	}
}

// sendCompleteError reports a shared error to complete callers.
func sendCompleteError(items []batchItem, err error) {
	if err == nil {
		err = errors.New("batch complete response mismatch")
	}
	for _, item := range items {
		item.errCh <- err
	}
}

// resetTimer safely resets a timer to the provided interval.
func resetTimer(timer *time.Timer, interval time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(interval)
}
