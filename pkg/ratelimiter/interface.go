package ratelimiter

import "context"

// Limiter is the client-facing API for reserve and complete operations.
type Limiter interface {
	Reserve(ctx context.Context, req ReserveRequest) (ReserveResponse, error)
	Complete(ctx context.Context, req CompleteRequest) (CompleteResponse, error)
	BatchReserve(ctx context.Context, req BatchReserveRequest) (BatchReserveResponse, error)
	BatchComplete(ctx context.Context, req BatchCompleteRequest) (BatchCompleteResponse, error)
}
