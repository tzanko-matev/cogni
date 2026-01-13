package memory

import (
	"context"

	"cogni/pkg/ratelimiter"
)

// ApplyDefinition creates or updates a limit definition.
func (m *MemoryBackend) ApplyDefinition(_ context.Context, def ratelimiter.LimitDefinition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	prev, ok := m.defs[def.Key]
	if !ok {
		m.defs[def.Key] = def
		m.states[def.Key] = ratelimiter.LimitState{Definition: def, Status: ratelimiter.LimitStatusActive}
		m.ensureLimitStoresLocked(def)
		m.updateCapacityLocked(def)
		return nil
	}

	if def.Capacity >= prev.Capacity {
		m.defs[def.Key] = def
		m.states[def.Key] = ratelimiter.LimitState{Definition: def, Status: ratelimiter.LimitStatusActive}
		m.ensureLimitStoresLocked(def)
		m.updateCapacityLocked(def)
		return nil
	}

	m.states[def.Key] = ratelimiter.LimitState{
		Definition:        prev,
		Status:            ratelimiter.LimitStatusDecreasing,
		PendingDecreaseTo: def.Capacity,
	}
	return nil
}

// TryApplyDecrease applies a pending capacity decrease when usage allows.
func (m *MemoryBackend) TryApplyDecrease(key ratelimiter.LimitKey) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.states[key]
	if !ok || state.Status != ratelimiter.LimitStatusDecreasing {
		return
	}

	current := state.Definition.Capacity
	target := state.PendingDecreaseTo
	if target == 0 || target >= current {
		return
	}

	m.cleanupLocked(key)
	available := m.availableCapacityLocked(key, state.Definition)
	if available < current-target {
		return
	}

	state.Definition.Capacity = target
	state.Status = ratelimiter.LimitStatusActive
	state.PendingDecreaseTo = 0
	m.states[key] = state
	m.defs[key] = state.Definition
	m.updateCapacityLocked(state.Definition)
}

func (m *MemoryBackend) ensureLimitStoresLocked(def ratelimiter.LimitDefinition) {
	switch def.Kind {
	case ratelimiter.KindRolling:
		if _, ok := m.roll[def.Key]; !ok {
			m.roll[def.Key] = newRollingLimit(def.Capacity)
		}
	case ratelimiter.KindConcurrency:
		if _, ok := m.conc[def.Key]; !ok {
			m.conc[def.Key] = newConcLimit(def.Capacity)
		}
	}
}

func (m *MemoryBackend) updateCapacityLocked(def ratelimiter.LimitDefinition) {
	switch def.Kind {
	case ratelimiter.KindRolling:
		if limit, ok := m.roll[def.Key]; ok {
			limit.cap = def.Capacity
		}
	case ratelimiter.KindConcurrency:
		if limit, ok := m.conc[def.Key]; ok {
			limit.cap = def.Capacity
		}
	}
}
