package runner

import (
	"bytes"
	"testing"
	"time"
)

// TestFormatRunID verifies run ID formatting.
func TestFormatRunID(t *testing.T) {
	timestamp := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	got := FormatRunID(timestamp, "deadbeef")
	if got != "20240102T030405Z-deadbeef" {
		t.Fatalf("unexpected run id: %q", got)
	}
}

// TestNewRunIDWithRand verifies deterministic run ID generation with a reader.
func TestNewRunIDWithRand(t *testing.T) {
	timestamp := time.Date(2024, 6, 7, 8, 9, 10, 0, time.UTC)
	reader := bytes.NewReader([]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55})
	got, err := NewRunIDWithRand(timestamp, reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "20240607T080910Z-001122334455" {
		t.Fatalf("unexpected run id: %q", got)
	}
}
