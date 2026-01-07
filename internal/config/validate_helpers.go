package config

import "strings"

// hasGlob reports whether a path includes glob characters.
func hasGlob(value string) bool {
	return strings.ContainsAny(value, "*?[]")
}
