package memory

import (
	"time"

	"cogni/pkg/ratelimiter"
)

func (m *MemoryBackend) now(at time.Time) time.Time {
	if !at.IsZero() {
		return at
	}
	return m.clock.Now()
}

func (m *MemoryBackend) cleanupLocked(key ratelimiter.LimitKey) {
	def, ok := m.defs[key]
	if !ok {
		return
	}
	switch def.Kind {
	case ratelimiter.KindRolling:
		if limit, ok := m.roll[key]; ok {
			cleanupRolling(limit, m.clock.Now())
		}
	case ratelimiter.KindConcurrency:
		if limit, ok := m.conc[key]; ok {
			cleanupConcurrency(limit, m.clock.Now())
		}
	}
}

func (m *MemoryBackend) availableCapacityLocked(key ratelimiter.LimitKey, def ratelimiter.LimitDefinition) uint64 {
	switch def.Kind {
	case ratelimiter.KindRolling:
		if limit, ok := m.roll[key]; ok {
			if limit.used >= limit.cap {
				return 0
			}
			return limit.cap - limit.used
		}
	case ratelimiter.KindConcurrency:
		if limit, ok := m.conc[key]; ok {
			used := uint64(len(limit.holds))
			if used >= limit.cap {
				return 0
			}
			return limit.cap - used
		}
	}
	return 0
}
