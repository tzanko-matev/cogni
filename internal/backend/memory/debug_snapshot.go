//go:build test || stress

package memory

import "cogni/pkg/ratelimiter"

// DebugSnapshot exposes internal counters for tests.
type DebugSnapshot struct {
	Rolling     map[ratelimiter.LimitKey]RollingSnapshot
	Concurrency map[ratelimiter.LimitKey]ConcurrencySnapshot
	Debt        map[ratelimiter.LimitKey]uint64
}

// RollingSnapshot captures rolling usage information.
type RollingSnapshot struct {
	Capacity uint64
	Used     uint64
}

// ConcurrencySnapshot captures concurrency usage information.
type ConcurrencySnapshot struct {
	Capacity uint64
	Holds    int
}

// DebugSnapshot returns a copy of in-memory counters.
func (m *MemoryBackend) DebugSnapshot() DebugSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	rolling := make(map[ratelimiter.LimitKey]RollingSnapshot, len(m.roll))
	for key, limit := range m.roll {
		rolling[key] = RollingSnapshot{Capacity: limit.cap, Used: limit.used}
	}
	conc := make(map[ratelimiter.LimitKey]ConcurrencySnapshot, len(m.conc))
	for key, limit := range m.conc {
		conc[key] = ConcurrencySnapshot{Capacity: limit.cap, Holds: len(limit.holds)}
	}
	debt := make(map[ratelimiter.LimitKey]uint64, len(m.debt))
	for key, value := range m.debt {
		debt[key] = value
	}

	return DebugSnapshot{Rolling: rolling, Concurrency: conc, Debt: debt}
}
