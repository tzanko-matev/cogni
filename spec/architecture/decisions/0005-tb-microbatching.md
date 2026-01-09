# ADR 0005: Microbatch TigerBeetle Transfer Submissions

## Status

- Proposed

## Context

- TigerBeetle clients allow only one in-flight request per session and have a max batch size.
- The rate limiter server will receive many concurrent Reserve/Complete requests.
- Linked transfer chains cannot be split across batches.

## Decision

- Implement a submitter goroutine that microbatches transfer work items:
  - Collect work items into batches up to a configured size or flush interval.
  - Never split a work item across batches to preserve linked chains.
  - Map TigerBeetle failure indices back to their originating work items.

## Specification

### Types

```go
// WorkItem represents a single logical operation (Reserve or Complete).
type WorkItem struct {
  Transfers []tb.Transfer
  Done      chan WorkResult
}

type WorkResult struct {
  Errors map[int]tb.CreateTransfersError // index -> error code
}

// TBSubmitter batches work and submits to TB.
type TBSubmitter struct {
  In          chan WorkItem
  FlushEvery  time.Duration
  MaxEvents   int
  ClientPool  *TBClientPool
}
```

### Algorithm (pseudo-code)

```go
func (s *TBSubmitter) Run(ctx context.Context) {
  var pending []WorkItem
  timer := time.NewTimer(s.FlushEvery)
  defer timer.Stop()

  flush := func() {
    if len(pending) == 0 {
      return
    }
    batch, mapping := buildBatch(pending, s.MaxEvents)
    results := s.ClientPool.Submit(batch)
    distribute(results, mapping, pending)
    pending = nil
  }

  for {
    select {
    case <-ctx.Done():
      flush()
      return
    case item := <-s.In:
      pending = append(pending, item)
      if batchSize(pending) >= s.MaxEvents {
        flush()
        resetTimer(timer, s.FlushEvery)
      }
    case <-timer.C:
      flush()
      resetTimer(timer, s.FlushEvery)
    }
  }
}
```

### Batch construction rules

- A WorkItem’s transfers must remain contiguous in the batch.
- If adding a WorkItem would exceed `MaxEvents`, flush first.
- Never split a linked chain across batches.

### Error mapping

- TB returns only failures with an index into the batch.
- The submitter builds a default `ok` status for all events, then fills errors for returned indices.
- Indices are mapped back to the originating WorkItem, and sent to its `Done` channel.

## Consequences

- Positive: Higher throughput and better TB utilization.
- Negative: Added complexity and potential latency jitter from batching.

## Alternatives considered

- One TB request per Reserve/Complete (rejected: low throughput).
- Splitting linked chains across batches (rejected: invalid TB semantics).
