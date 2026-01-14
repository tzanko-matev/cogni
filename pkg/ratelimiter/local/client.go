package local

import (
	"context"
	"fmt"
	"time"

	"cogni/internal/backend/memory"
	"cogni/internal/registry"
	"cogni/pkg/ratelimiter"
)

// Client implements Limiter using the in-memory backend.
type Client struct {
	backend *memory.MemoryBackend
	now     func() time.Time
}

// NewMemoryLimiterFromFile loads limits from disk and returns a local client.
func NewMemoryLimiterFromFile(path string) (*Client, error) {
	reg := registry.New()
	if err := reg.Load(path); err != nil {
		return nil, err
	}
	return NewMemoryLimiterFromStates(reg.List())
}

// NewMemoryLimiterFromStates loads limits from memory and returns a local client.
func NewMemoryLimiterFromStates(states []ratelimiter.LimitState) (*Client, error) {
	backend := memory.New(nil)
	for _, state := range states {
		if err := backend.ApplyState(state); err != nil {
			return nil, fmt.Errorf("apply limit state: %w", err)
		}
	}
	return &Client{backend: backend, now: time.Now}, nil
}

// Reserve forwards reserve requests to the backend.
func (c *Client) Reserve(ctx context.Context, req ratelimiter.ReserveRequest) (ratelimiter.ReserveResponse, error) {
	return c.backend.Reserve(ctx, req, c.now())
}

// Complete forwards completion requests to the backend.
func (c *Client) Complete(ctx context.Context, req ratelimiter.CompleteRequest) (ratelimiter.CompleteResponse, error) {
	return c.backend.Complete(ctx, req)
}

// BatchReserve executes batch reserve requests locally.
func (c *Client) BatchReserve(ctx context.Context, req ratelimiter.BatchReserveRequest) (ratelimiter.BatchReserveResponse, error) {
	results := make([]ratelimiter.BatchReserveResult, 0, len(req.Requests))
	for _, item := range req.Requests {
		res, err := c.backend.Reserve(ctx, item, c.now())
		if err != nil {
			results = append(results, ratelimiter.BatchReserveResult{Allowed: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchReserveResult{
			Allowed:        res.Allowed,
			RetryAfterMs:   res.RetryAfterMs,
			ReservedAtUnix: res.ReservedAtUnixMs,
			Error:          res.Error,
		})
	}
	return ratelimiter.BatchReserveResponse{Results: results}, nil
}

// BatchComplete executes batch complete requests locally.
func (c *Client) BatchComplete(ctx context.Context, req ratelimiter.BatchCompleteRequest) (ratelimiter.BatchCompleteResponse, error) {
	results := make([]ratelimiter.BatchCompleteResult, 0, len(req.Requests))
	for _, item := range req.Requests {
		res, err := c.backend.Complete(ctx, item)
		if err != nil {
			results = append(results, ratelimiter.BatchCompleteResult{Ok: false, Error: "backend_error"})
			continue
		}
		results = append(results, ratelimiter.BatchCompleteResult{Ok: res.Ok, Error: res.Error})
	}
	return ratelimiter.BatchCompleteResponse{Results: results}, nil
}
