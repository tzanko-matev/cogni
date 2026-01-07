package tools

import (
	"io"
	"os"
)

// fileSystem abstracts filesystem access for tool runners.
type fileSystem interface {
	Open(name string) (io.ReadCloser, error)
	Stat(name string) (os.FileInfo, error)
}

// osFileSystem implements fileSystem using the OS.
type osFileSystem struct{}

// Open opens a file for reading.
func (osFileSystem) Open(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

// Stat returns file info for a path.
func (osFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
