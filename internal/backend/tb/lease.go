package tb

import "cogni/pkg/ratelimiter"

// LeaseState stores reservation metadata for reconciliation.
type LeaseState struct {
	LeaseID         string
	ReservedAtUnix  int64
	Requirements    []ratelimiter.Requirement
	ReservedAmounts map[ratelimiter.LimitKey]uint64
}

// indexByKey maps requirements to amounts.
func indexByKey(reqs []ratelimiter.Requirement) map[ratelimiter.LimitKey]uint64 {
	out := make(map[ratelimiter.LimitKey]uint64, len(reqs))
	for _, req := range reqs {
		out[req.Key] = req.Amount
	}
	return out
}

// requirementsEqual checks for equality of requirement sets.
func requirementsEqual(a, b []ratelimiter.Requirement) bool {
	if len(a) != len(b) {
		return false
	}
	lookup := make(map[ratelimiter.LimitKey]uint64, len(a))
	for _, req := range a {
		lookup[req.Key] = req.Amount
	}
	for _, req := range b {
		amount, ok := lookup[req.Key]
		if !ok || amount != req.Amount {
			return false
		}
	}
	return true
}
