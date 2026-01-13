package testutil

import (
	"testing"
	"time"
)

// Eventually polls fn until it returns true or timeout elapses.
func Eventually(t *testing.T, timeout, interval time.Duration, fn func() bool, msg string) {
	t.Helper()
	deadline := time.After(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if fn() {
			return
		}
		select {
		case <-deadline:
			if msg == "" {
				t.Fatalf("condition not met before timeout")
			}
			t.Fatalf("%s", msg)
		case <-ticker.C:
		}
	}
}
