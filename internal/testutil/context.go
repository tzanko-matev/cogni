package testutil

import (
	"context"
	"testing"
	"time"
)

// DefaultTimeout is the standard timeout for unit tests.
const DefaultTimeout = 5 * time.Second

// Context returns a context with timeout tied to the test lifecycle.
func Context(t testing.TB, timeout time.Duration) context.Context {
	t.Helper()
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if deadline, ok := t.Deadline(); ok {
		remaining := time.Until(deadline) - time.Second
		if remaining > 0 && remaining < timeout {
			timeout = remaining
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}
