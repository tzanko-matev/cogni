package memory

import (
	"sync"

	"cogni/internal/registry"
	"cogni/pkg/ratelimiter"
)

// MemoryBackend stores rate limiter state in memory.
type MemoryBackend struct {
	mu           sync.Mutex
	clock        Clock
	registry     *registry.Registry
	registryPath string
	defs         map[ratelimiter.LimitKey]ratelimiter.LimitDefinition
	states       map[ratelimiter.LimitKey]ratelimiter.LimitState
	roll         map[ratelimiter.LimitKey]*rollingLimit
	conc         map[ratelimiter.LimitKey]*concLimit
	debt         map[ratelimiter.LimitKey]uint64
	leases       map[string]LeaseState
}

// New creates a MemoryBackend with the provided clock.
func New(clock Clock) *MemoryBackend {
	if clock == nil {
		clock = realClock{}
	}
	return &MemoryBackend{
		clock:  clock,
		defs:   map[ratelimiter.LimitKey]ratelimiter.LimitDefinition{},
		states: map[ratelimiter.LimitKey]ratelimiter.LimitState{},
		roll:   map[ratelimiter.LimitKey]*rollingLimit{},
		conc:   map[ratelimiter.LimitKey]*concLimit{},
		debt:   map[ratelimiter.LimitKey]uint64{},
		leases: map[string]LeaseState{},
	}
}
