package runner

import (
	"io"
	"sync"
)

// lockedWriter serializes writes to an underlying writer.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write writes to the underlying writer with a mutex guard.
func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// wrapVerboseWriters returns concurrency-safe writers when workers > 1.
func wrapVerboseWriters(workers int, verboseWriter io.Writer, verboseLogWriter io.Writer) (io.Writer, io.Writer) {
	if workers <= 1 {
		return verboseWriter, verboseLogWriter
	}
	if verboseWriter != nil {
		verboseWriter = &lockedWriter{w: verboseWriter}
	}
	if verboseLogWriter != nil {
		verboseLogWriter = &lockedWriter{w: verboseLogWriter}
	}
	return verboseWriter, verboseLogWriter
}
