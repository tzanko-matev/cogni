package runner

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"time"
)

const runIDSuffixBytes = 6

func NewRunID() (string, error) {
	return NewRunIDWithRand(time.Now().UTC(), rand.Reader)
}

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

func FormatRunID(now time.Time, suffix string) string {
	return now.UTC().Format("20060102T150405Z") + "-" + suffix
}
