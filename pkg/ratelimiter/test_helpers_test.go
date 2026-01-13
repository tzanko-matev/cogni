package ratelimiter

import (
	"testing"
	"time"

	"cogni/internal/testutil"
)

// runWithTimeout fails the test if fn does not complete within timeout.
func runWithTimeout(t *testing.T, timeout time.Duration, fn func()) {
	t.Helper()
	ctx := testutil.Context(t, timeout)
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-ctx.Done():
		t.Fatalf("test timed out")
	case <-done:
	}
}

// waitFor waits for a signal on ch or fails after timeout.
func waitFor(t *testing.T, ch <-chan struct{}, timeout time.Duration) {
	t.Helper()
	ctx := testutil.Context(t, timeout)
	select {
	case <-ctx.Done():
		t.Fatalf("timeout waiting for signal")
	case <-ch:
	}
}

// waitForCount waits for count signals or fails after timeout.
func waitForCount(t *testing.T, ch <-chan struct{}, count int, timeout time.Duration) {
	t.Helper()
	if count <= 0 {
		return
	}
	ctx := testutil.Context(t, timeout)
	seen := 0
	for seen < count {
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for %d signals (got %d)", count, seen)
		case <-ch:
			seen++
		}
	}
}
