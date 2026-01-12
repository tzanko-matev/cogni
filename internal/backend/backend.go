package backend

import (
	"context"
	"time"

	"cogni/pkg/ratelimiter"
)

// Backend provides server-side rate limiter operations.
type Backend interface {
	ApplyDefinition(ctx context.Context, def ratelimiter.LimitDefinition) error
	Reserve(ctx context.Context, req ratelimiter.ReserveRequest, now time.Time) (ratelimiter.ReserveResponse, error)
	Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error)
}
