package tools

import (
	"io"
	"os"
)

// fileSystem abstracts filesystem access for tool runners.
type fileSystem interface {
	Open(name string) (io.ReadCloser, error)
	Stat(name string) (os.FileInfo, error)
	Lstat(name string) (os.FileInfo, error)
	ReadDir(name string) ([]os.DirEntry, error)
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

// Lstat returns file info without following symlinks.
func (osFileSystem) Lstat(name string) (os.FileInfo, error) {
	return os.Lstat(name)
}

// ReadDir reads directory entries.
func (osFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}
