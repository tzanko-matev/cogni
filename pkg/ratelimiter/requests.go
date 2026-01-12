package ratelimiter

// ReserveRequest asks to reserve capacity for a lease.
type ReserveRequest struct {
	LeaseID      string        `json:"lease_id"`
	JobID        string        `json:"job_id"`
	Requirements []Requirement `json:"requirements"`
}

// ReserveResponse reports whether a reservation was allowed.
type ReserveResponse struct {
	Allowed          bool   `json:"allowed"`
	RetryAfterMs     int    `json:"retry_after_ms"`
	ReservedAtUnixMs int64  `json:"reserved_at_unix_ms"`
	Error            string `json:"error"`
}

// CompleteRequest reports actual usage for a lease.
type CompleteRequest struct {
	LeaseID string   `json:"lease_id"`
	JobID   string   `json:"job_id"`
	Actuals []Actual `json:"actuals"`
}

// CompleteResponse reports whether completion succeeded.
type CompleteResponse struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// BatchReserveRequest batches multiple reserve requests.
type BatchReserveRequest struct {
	Requests []ReserveRequest `json:"requests"`
}

// BatchReserveResult reports a single reserve outcome in a batch.
type BatchReserveResult struct {
	Allowed        bool   `json:"allowed"`
	RetryAfterMs   int    `json:"retry_after_ms"`
	ReservedAtUnix int64  `json:"reserved_at_unix_ms"`
	Error          string `json:"error"`
}

// BatchReserveResponse aggregates reserve results in order.
type BatchReserveResponse struct {
	Results []BatchReserveResult `json:"results"`
}

// BatchCompleteRequest batches multiple complete requests.
type BatchCompleteRequest struct {
	Requests []CompleteRequest `json:"requests"`
}

// BatchCompleteResult reports a single complete outcome in a batch.
type BatchCompleteResult struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}

// BatchCompleteResponse aggregates complete results in order.
type BatchCompleteResponse struct {
	Results []BatchCompleteResult `json:"results"`
}
