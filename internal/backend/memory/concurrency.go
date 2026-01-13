package memory

import (
	"container/heap"
	"time"
)

type concLimit struct {
	cap   uint64
	holds map[string]time.Time
	heap  concHeap
}

type concHold struct {
	id        string
	expiresAt time.Time
	heapIndex int
}

type concHeap []*concHold

func (h concHeap) Len() int { return len(h) }

func (h concHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h concHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h *concHeap) Push(x interface{}) {
	hold := x.(*concHold)
	hold.heapIndex = len(*h)
	*h = append(*h, hold)
}

func (h *concHeap) Pop() interface{} {
	old := *h
	n := len(old)
	hold := old[n-1]
	hold.heapIndex = -1
	*h = old[:n-1]
	return hold
}

func newConcLimit(capacity uint64) *concLimit {
	return &concLimit{
		cap:   capacity,
		holds: map[string]time.Time{},
		heap:  concHeap{},
	}
}

func cleanupConcurrency(limit *concLimit, now time.Time) {
	for limit.heap.Len() > 0 {
		hold := limit.heap[0]
		if hold.expiresAt.After(now) {
			break
		}
		heap.Pop(&limit.heap)
		if _, ok := limit.holds[hold.id]; !ok {
			continue
		}
		delete(limit.holds, hold.id)
	}
}

func addConcurrencyHold(limit *concLimit, leaseID string, expiresAt time.Time) {
	limit.holds[leaseID] = expiresAt
	heap.Push(&limit.heap, &concHold{id: leaseID, expiresAt: expiresAt})
}
