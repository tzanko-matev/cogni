package httpclient

import (
	"context"
	"testing"
	"time"
)

// TestNewWithTimeoutSetsTimeout ensures the HTTP client timeout is applied.
func TestNewWithTimeoutSetsTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	t.Cleanup(cancel)
	done := make(chan struct{})
	go func() {
		defer close(done)
		timeout := 1500 * time.Millisecond
		client := NewWithTimeout("http://example", timeout)
		if client.client.Timeout != timeout {
			t.Fatalf("expected timeout %s, got %s", timeout, client.client.Timeout)
		}
	}()
	select {
	case <-done:
	case <-ctx.Done():
		t.Fatalf("test timed out")
	}
}
