package memory

import (
	"context"
	"time"

	"cogni/pkg/ratelimiter"
)

const invalidRequestError = "invalid_request"

// Reserve reserves capacity for the requested requirements.
func (m *MemoryBackend) Reserve(_ context.Context, req ratelimiter.ReserveRequest, at time.Time) (ratelimiter.ReserveResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if req.LeaseID == "" || len(req.Requirements) == 0 {
		return ratelimiter.ReserveResponse{Allowed: false, Error: invalidRequestError}, nil
	}
	if state, ok := m.leases[req.LeaseID]; ok {
		if requirementsEqual(state.Requirements, req.Requirements) {
			return ratelimiter.ReserveResponse{
				Allowed:          true,
				ReservedAtUnixMs: state.ReservedAtUnix,
			}, nil
		}
		return ratelimiter.ReserveResponse{Allowed: false, Error: invalidRequestError}, nil
	}

	for _, r := range req.Requirements {
		state, ok := m.states[r.Key]
		if ok && state.Status == ratelimiter.LimitStatusDecreasing {
			return ratelimiter.ReserveResponse{
				Allowed:      false,
				RetryAfterMs: decreaseRetryMs,
				Error:        "limit_decreasing:" + string(r.Key),
			}, nil
		}
	}
	for _, r := range req.Requirements {
		if _, ok := m.defs[r.Key]; !ok {
			return ratelimiter.ReserveResponse{Allowed: false, Error: "unknown_limit_key:" + string(r.Key)}, nil
		}
	}

	now := m.now(at)
	for _, r := range req.Requirements {
		def := m.defs[r.Key]
		switch def.Kind {
		case ratelimiter.KindRolling:
			cleanupRolling(m.roll[r.Key], now)
		case ratelimiter.KindConcurrency:
			cleanupConcurrency(m.conc[r.Key], now)
		}
	}

	maxRetry := 0
	for _, r := range req.Requirements {
		def := m.defs[r.Key]
		switch def.Kind {
		case ratelimiter.KindRolling:
			if m.roll[r.Key].used+r.Amount > m.roll[r.Key].cap {
				maxRetry = maxInt(maxRetry, retryAfter(def))
			}
		case ratelimiter.KindConcurrency:
			if uint64(len(m.conc[r.Key].holds)+1) > m.conc[r.Key].cap {
				maxRetry = maxInt(maxRetry, retryAfter(def))
			}
		}
	}
	if maxRetry > 0 {
		return ratelimiter.ReserveResponse{Allowed: false, RetryAfterMs: maxRetry}, nil
	}

	for _, r := range req.Requirements {
		def := m.defs[r.Key]
		switch def.Kind {
		case ratelimiter.KindRolling:
			addRollingReservation(m.roll[r.Key], req.LeaseID, r.Amount, now.Add(time.Duration(def.WindowSeconds)*time.Second))
		case ratelimiter.KindConcurrency:
			addConcurrencyHold(m.conc[r.Key], req.LeaseID, now.Add(time.Duration(def.TimeoutSeconds)*time.Second))
		}
	}

	m.leases[req.LeaseID] = LeaseState{
		LeaseID:         req.LeaseID,
		ReservedAtUnix:  now.UnixMilli(),
		Requirements:    req.Requirements,
		ReservedAmounts: indexByKey(req.Requirements),
	}

	return ratelimiter.ReserveResponse{Allowed: true, ReservedAtUnixMs: now.UnixMilli()}, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
