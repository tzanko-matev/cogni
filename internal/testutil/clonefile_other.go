//go:build !darwin

package testutil

import "errors"

// cloneFile reports that copy-on-write cloning is unavailable on this platform.
func cloneFile(src, dst string) error {
	return errors.New("clonefile not supported")
}
