package api

import (
	"errors"

	"cogni/pkg/ratelimiter"
)

var errInvalidDefinition = errors.New("invalid limit definition")

func ratelimiterKey(value string) ratelimiter.LimitKey {
	return ratelimiter.LimitKey(value)
}
