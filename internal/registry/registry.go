package registry

import (
	"sort"
	"sync"

	"cogni/pkg/ratelimiter"
)

// Registry stores limit states keyed by limit key.
type Registry struct {
	mu     sync.RWMutex
	states map[ratelimiter.LimitKey]ratelimiter.LimitState
}

// New creates an empty registry.
func New() *Registry {
	return &Registry{states: map[ratelimiter.LimitKey]ratelimiter.LimitState{}}
}

// Get returns the limit state for a key, if present.
func (r *Registry) Get(key ratelimiter.LimitKey) (ratelimiter.LimitState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.states[key]
	return state, ok
}

// Put inserts or replaces a limit state.
func (r *Registry) Put(state ratelimiter.LimitState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.states[state.Definition.Key] = state
}

// List returns all limit states sorted by key.
func (r *Registry) List() []ratelimiter.LimitState {
	r.mu.RLock()
	snapshot := make([]ratelimiter.LimitState, 0, len(r.states))
	for _, state := range r.states {
		snapshot = append(snapshot, state)
	}
	r.mu.RUnlock()
	sort.Slice(snapshot, func(i, j int) bool {
		return snapshot[i].Definition.Key < snapshot[j].Definition.Key
	})
	return snapshot
}

// NextState computes the resulting state for a definition update without mutating.
func (r *Registry) NextState(def ratelimiter.LimitDefinition) ratelimiter.LimitState {
	r.mu.RLock()
	prev, ok := r.states[def.Key]
	r.mu.RUnlock()
	return nextState(prev, ok, def)
}

func nextState(prev ratelimiter.LimitState, ok bool, def ratelimiter.LimitDefinition) ratelimiter.LimitState {
	if !ok {
		return ratelimiter.LimitState{
			Definition:        def,
			Status:            ratelimiter.LimitStatusActive,
			PendingDecreaseTo: 0,
		}
	}
	if def.Capacity >= prev.Definition.Capacity {
		return ratelimiter.LimitState{
			Definition:        def,
			Status:            ratelimiter.LimitStatusActive,
			PendingDecreaseTo: 0,
		}
	}
	return ratelimiter.LimitState{
		Definition:        prev.Definition,
		Status:            ratelimiter.LimitStatusDecreasing,
		PendingDecreaseTo: def.Capacity,
	}
}
