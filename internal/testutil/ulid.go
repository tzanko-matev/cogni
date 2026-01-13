package testutil

import "cogni/pkg/ratelimiter"

// NewULID returns a ULID string for use in tests.
func NewULID() string {
	return ratelimiter.NewULID()
}
