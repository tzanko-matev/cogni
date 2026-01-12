package api

import "cogni/pkg/ratelimiter"

type limitResponse struct {
	Limit ratelimiter.LimitState `json:"limit"`
}

type limitsResponse struct {
	Limits []ratelimiter.LimitState `json:"limits"`
}
