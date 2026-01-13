package memory

import (
	"container/heap"
	"time"
)

type rollingLimit struct {
	cap  uint64
	used uint64
	heap reservationHeap
	byID map[string]*reservation
}

type reservation struct {
	id        string
	amount    uint64
	expiresAt time.Time
	heapIndex int
}

type reservationHeap []*reservation

func (h reservationHeap) Len() int { return len(h) }

func (h reservationHeap) Less(i, j int) bool {
	return h[i].expiresAt.Before(h[j].expiresAt)
}

func (h reservationHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].heapIndex = i
	h[j].heapIndex = j
}

func (h *reservationHeap) Push(x interface{}) {
	res := x.(*reservation)
	res.heapIndex = len(*h)
	*h = append(*h, res)
}

func (h *reservationHeap) Pop() interface{} {
	old := *h
	n := len(old)
	res := old[n-1]
	res.heapIndex = -1
	*h = old[:n-1]
	return res
}

func newRollingLimit(capacity uint64) *rollingLimit {
	return &rollingLimit{
		cap:  capacity,
		used: 0,
		heap: reservationHeap{},
		byID: map[string]*reservation{},
	}
}

func cleanupRolling(limit *rollingLimit, now time.Time) {
	for limit.heap.Len() > 0 {
		res := limit.heap[0]
		if res.expiresAt.After(now) {
			break
		}
		heap.Pop(&limit.heap)
		delete(limit.byID, res.id)
		if limit.used >= res.amount {
			limit.used -= res.amount
		} else {
			limit.used = 0
		}
	}
}

func addRollingReservation(limit *rollingLimit, leaseID string, amount uint64, expiresAt time.Time) {
	res := &reservation{id: leaseID, amount: amount, expiresAt: expiresAt}
	limit.byID[leaseID] = res
	limit.used += amount
	heap.Push(&limit.heap, res)
}

func reduceRollingReservation(limit *rollingLimit, leaseID string, newAmount uint64) {
	res, ok := limit.byID[leaseID]
	if !ok {
		return
	}
	if newAmount >= res.amount {
		return
	}
	diff := res.amount - newAmount
	if limit.used >= diff {
		limit.used -= diff
	} else {
		limit.used = 0
	}
	res.amount = newAmount
}
