package memory

import "cogni/pkg/ratelimiter"

// LeaseState caches reservation metadata for reconciliation.
type LeaseState struct {
	LeaseID         string
	ReservedAtUnix  int64
	Requirements    []ratelimiter.Requirement
	ReservedAmounts map[ratelimiter.LimitKey]uint64
}

func indexByKey(reqs []ratelimiter.Requirement) map[ratelimiter.LimitKey]uint64 {
	out := make(map[ratelimiter.LimitKey]uint64, len(reqs))
	for _, req := range reqs {
		out[req.Key] = req.Amount
	}
	return out
}

func requirementsEqual(a, b []ratelimiter.Requirement) bool {
	if len(a) != len(b) {
		return false
	}
	lookup := make(map[ratelimiter.LimitKey]uint64, len(a))
	for _, req := range a {
		lookup[req.Key] = req.Amount
	}
	for _, req := range b {
		if amount, ok := lookup[req.Key]; !ok || amount != req.Amount {
			return false
		}
	}
	return true
}
