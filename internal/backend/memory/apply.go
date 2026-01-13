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

// ApplyState loads a persisted limit state into memory.
func (m *MemoryBackend) ApplyState(state ratelimiter.LimitState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defs[state.Definition.Key] = state.Definition
	m.states[state.Definition.Key] = state
	m.ensureLimitStoresLocked(state.Definition)
	m.updateCapacityLocked(state.Definition)
	return nil
}

// TryApplyDecrease applies a pending capacity decrease when usage allows.
func (m *MemoryBackend) TryApplyDecrease(key ratelimiter.LimitKey) {
	var (
		registryPath string
		appliedState ratelimiter.LimitState
		applied      bool
	)

	m.mu.Lock()
	state, ok := m.states[key]
	if !ok || state.Status != ratelimiter.LimitStatusDecreasing {
		m.mu.Unlock()
		return
	}

	current := state.Definition.Capacity
	target := state.PendingDecreaseTo
	if target == 0 || target >= current {
		m.mu.Unlock()
		return
	}

	m.cleanupLocked(key)
	available := m.availableCapacityLocked(key, state.Definition)
	if available < current-target {
		m.mu.Unlock()
		return
	}

	state.Definition.Capacity = target
	state.Status = ratelimiter.LimitStatusActive
	state.PendingDecreaseTo = 0
	m.states[key] = state
	m.defs[key] = state.Definition
	m.updateCapacityLocked(state.Definition)
	appliedState = state
	registryPath = m.registryPath
	applied = m.registry != nil
	reg := m.registry
	m.mu.Unlock()

	if applied {
		reg.Put(appliedState)
		if registryPath != "" {
			_ = reg.Save(registryPath)
		}
		return
	}
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
