package api

import "cogni/pkg/ratelimiter"

const (
	maxRequirementsPerReserve = 32
	maxBatchRequests          = 256
	invalidRequestError       = "invalid_request"
	decreaseRetryMs           = 10000
)

type validationStatus int

const (
	validationInvalid validationStatus = iota
	validationDenied
	validationOK
)

type reserveValidationResult struct {
	status   validationStatus
	response ratelimiter.ReserveResponse
}

func (h *handler) validateReserve(req ratelimiter.ReserveRequest) reserveValidationResult {
	if req.LeaseID == "" {
		return reserveValidationResult{status: validationInvalid}
	}
	if len(req.Requirements) == 0 || len(req.Requirements) > maxRequirementsPerReserve {
		return reserveValidationResult{status: validationInvalid}
	}
	for _, r := range req.Requirements {
		if r.Key == "" || r.Amount == 0 {
			return reserveValidationResult{status: validationInvalid}
		}
		state, ok := h.registry.Get(r.Key)
		if !ok {
			return reserveValidationResult{
				status: validationDenied,
				response: ratelimiter.ReserveResponse{
					Allowed: false,
					Error:   "unknown_limit_key:" + string(r.Key),
				},
			}
		}
		if state.Status == ratelimiter.LimitStatusDecreasing {
			return reserveValidationResult{
				status: validationDenied,
				response: ratelimiter.ReserveResponse{
					Allowed:      false,
					RetryAfterMs: decreaseRetryMs,
					Error:        "limit_decreasing:" + string(r.Key),
				},
			}
		}
	}
	return reserveValidationResult{status: validationOK}
}
