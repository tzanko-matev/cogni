package eval

import "os"

// QADeps supplies external dependencies for QA evaluation.
type QADeps struct {
	FS QAFileSystem
}

// QAFileSystem abstracts filesystem reads for citation validation.
type QAFileSystem interface {
	Stat(name string) (os.FileInfo, error)
	ReadFile(name string) ([]byte, error)
}

// osQAFileSystem uses the OS filesystem for QA evaluation.
type osQAFileSystem struct{}

// Stat returns file info for a path.
func (osQAFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

// ReadFile reads a file from disk.
func (osQAFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
