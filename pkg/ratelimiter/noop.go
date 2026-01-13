package ratelimiter

import "context"

// NoopLimiter is a Limiter implementation that always allows requests.
var NoopLimiter Limiter = noopLimiter{}

// noopLimiter satisfies Limiter without enforcing limits.
type noopLimiter struct{}

// Reserve accepts every reservation.
func (noopLimiter) Reserve(_ context.Context, _ ReserveRequest) (ReserveResponse, error) {
	return ReserveResponse{Allowed: true}, nil
}

// Complete accepts every completion.
func (noopLimiter) Complete(_ context.Context, _ CompleteRequest) (CompleteResponse, error) {
	return CompleteResponse{Ok: true}, nil
}

// BatchReserve accepts every batch reservation.
func (noopLimiter) BatchReserve(_ context.Context, req BatchReserveRequest) (BatchReserveResponse, error) {
	results := make([]BatchReserveResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, BatchReserveResult{Allowed: true})
	}
	return BatchReserveResponse{Results: results}, nil
}

// BatchComplete accepts every batch completion.
func (noopLimiter) BatchComplete(_ context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error) {
	results := make([]BatchCompleteResult, 0, len(req.Requests))
	for range req.Requests {
		results = append(results, BatchCompleteResult{Ok: true})
	}
	return BatchCompleteResponse{Results: results}, nil
}
