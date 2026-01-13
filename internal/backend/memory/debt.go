package memory

import "cogni/pkg/ratelimiter"

// DebtForKey returns recorded debt for a limit key.
func (m *MemoryBackend) DebtForKey(key ratelimiter.LimitKey) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.debt[key]
}
