//go:build darwin

package testutil

import "golang.org/x/sys/unix"

// cloneFile performs a copy-on-write clone on macOS when supported.
func cloneFile(src, dst string) error {
	return unix.Clonefile(src, dst, 0)
}
