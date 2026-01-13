package tb

import (
	"sync"

	"cogni/pkg/ratelimiter"
)

// denyTracker tracks denial streaks per limit key.
type denyTracker struct {
	mu     sync.Mutex
	counts map[ratelimiter.LimitKey]int
}

// newDenyTracker initializes an empty denial tracker.
func newDenyTracker() *denyTracker {
	return &denyTracker{counts: map[ratelimiter.LimitKey]int{}}
}

// Increment bumps the denial streak for a key.
func (d *denyTracker) Increment(key ratelimiter.LimitKey) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.counts[key]++
	return d.counts[key]
}

// Decay resets the denial streak for a key.
func (d *denyTracker) Decay(key ratelimiter.LimitKey) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.counts[key] = 0
	return 0
}
