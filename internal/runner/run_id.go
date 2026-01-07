package runner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

// runIDSuffixBytes controls the random suffix length for run IDs.
const runIDSuffixBytes = 6

// NewRunID generates a run identifier using the current time and crypto RNG.
func NewRunID() (string, error) {
	return NewRunIDWithRand(time.Now().UTC(), rand.Reader)
}

// NewRunIDWithRand generates a run identifier using an injected reader.
func NewRunIDWithRand(now time.Time, r io.Reader) (string, error) {
	if r == nil {
		return "", fmt.Errorf("random reader is nil")
	}
	buf := make([]byte, runIDSuffixBytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	suffix := hex.EncodeToString(buf)
	return FormatRunID(now, suffix), nil
}

// FormatRunID formats a run identifier from a timestamp and suffix.
func FormatRunID(now time.Time, suffix string) string {
	return now.UTC().Format("20060102T150405Z") + "-" + suffix
}
