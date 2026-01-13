package memory

import (
	"context"

	"cogni/pkg/ratelimiter"
)

// Complete reconciles reservations with actual usage.
func (m *MemoryBackend) Complete(_ context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, ok := m.leases[req.LeaseID]
	if !ok {
		return ratelimiter.CompleteResponse{Ok: true}, nil
	}

	for _, r := range state.Requirements {
		def, ok := m.defs[r.Key]
		if !ok {
			continue
		}
		if def.Kind == ratelimiter.KindConcurrency {
			if limit, ok := m.conc[r.Key]; ok {
				delete(limit.holds, req.LeaseID)
			}
		}
	}

	for _, actual := range req.Actuals {
		def, ok := m.defs[actual.Key]
		if !ok || def.Kind != ratelimiter.KindRolling {
			continue
		}
		reserved := state.ReservedAmounts[actual.Key]
		if actual.ActualAmount < reserved {
			reduceRollingReservation(m.roll[actual.Key], req.LeaseID, actual.ActualAmount)
			continue
		}
		if actual.ActualAmount > reserved && def.Overage == ratelimiter.OverageDebt {
			m.debt[actual.Key] += actual.ActualAmount - reserved
		}
	}

	delete(m.leases, req.LeaseID)
	return ratelimiter.CompleteResponse{Ok: true}, nil
}
