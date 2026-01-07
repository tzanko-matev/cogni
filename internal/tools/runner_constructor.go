package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DefaultLimits returns the default read/output limits.
func DefaultLimits() Limits {
	return Limits{
		MaxReadBytes:   200 * 1024,
		MaxOutputBytes: 200 * 1024,
	}
}

// NewRunner constructs a Runner for a repository root.
func NewRunner(root string) (*Runner, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("root is empty")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat root: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("root is not a directory")
	}
	return &Runner{
		Root:     abs,
		Limits:   DefaultLimits(),
		clock:    time.Now,
		rgRunner: execRGRunner{},
		fs:       osFileSystem{},
	}, nil
}
