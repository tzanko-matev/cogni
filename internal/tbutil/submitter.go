package tbutil

import (
	"context"
	"fmt"
	"time"

	tbtypes "github.com/tigerbeetle/tigerbeetle-go/pkg/types"
)

// WorkItem represents a single TB transfer request.
type WorkItem struct {
	Transfers []tbtypes.Transfer
	Done      chan WorkResult
}

// WorkResult reports per-transfer errors for a WorkItem.
type WorkResult struct {
	Errors map[int]tbtypes.CreateTransferResult
	Err    error
}

// Submitter batches transfer work items and submits them to TigerBeetle.
type Submitter struct {
	In         chan WorkItem
	FlushEvery time.Duration
	MaxEvents  int
	Pool       *ClientPool
}

// Run processes work items until the context is canceled.
func (s *Submitter) Run(ctx context.Context) {
	timer := time.NewTimer(s.FlushEvery)
	defer timer.Stop()
	var pending []WorkItem

	flush := func() {
		if len(pending) == 0 {
			return
		}
		items := pending
		pending = nil
		s.flush(ctx, items)
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case item := <-s.In:
			if len(item.Transfers) > s.MaxEvents {
				s.respond(item, WorkResult{Err: fmt.Errorf("work item exceeds max batch size")})
				continue
			}
			if len(pending) > 0 && totalTransfers(pending)+len(item.Transfers) > s.MaxEvents {
				flush()
				resetTimer(timer, s.FlushEvery)
			}
			pending = append(pending, item)
			if totalTransfers(pending) >= s.MaxEvents {
				flush()
				resetTimer(timer, s.FlushEvery)
			}
		case <-timer.C:
			flush()
			resetTimer(timer, s.FlushEvery)
		}
	}
}

// flush builds and submits a batch from work items.
func (s *Submitter) flush(ctx context.Context, items []WorkItem) {
	transfers := make([]tbtypes.Transfer, 0, totalTransfers(items))
	spans := make([]workSpan, 0, len(items))
	cursor := 0
	for _, item := range items {
		spans = append(spans, workSpan{item: item, start: cursor, length: len(item.Transfers)})
		transfers = append(transfers, item.Transfers...)
		cursor += len(item.Transfers)
	}

	results, err := s.submit(ctx, transfers)
	if err != nil {
		for _, span := range spans {
			s.respond(span.item, WorkResult{Err: err})
		}
		return
	}

	errorMap := map[int]tbtypes.CreateTransferResult{}
	for _, result := range results {
		errorMap[int(result.Index)] = result.Result
	}

	for _, span := range spans {
		itemErrors := map[int]tbtypes.CreateTransferResult{}
		for i := 0; i < span.length; i++ {
			index := span.start + i
			if result, ok := errorMap[index]; ok {
				itemErrors[i] = result
			}
		}
		s.respond(span.item, WorkResult{Errors: itemErrors})
	}
}

// submit performs the CreateTransfers call using a pooled client.
func (s *Submitter) submit(ctx context.Context, transfers []tbtypes.Transfer) ([]tbtypes.TransferEventResult, error) {
	client, err := s.Pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer s.Pool.Release(client)
	return CreateTransfers(ctx, client, transfers)
}

// respond sends a work result without blocking the submitter.
func (s *Submitter) respond(item WorkItem, result WorkResult) {
	select {
	case item.Done <- result:
	default:
	}
}

// workSpan tracks a work item's range within a batch.
type workSpan struct {
	item   WorkItem
	start  int
	length int
}

// totalTransfers counts the transfers across all pending items.
func totalTransfers(items []WorkItem) int {
	total := 0
	for _, item := range items {
		total += len(item.Transfers)
	}
	return total
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
