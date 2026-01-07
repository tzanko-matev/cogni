package cucumber

import "strings"

// tagExpression builds a godog tag expression from tags.
func tagExpression(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	parts := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if !strings.HasPrefix(tag, "@") && !strings.HasPrefix(tag, "~") {
			tag = "@" + tag
		}
		parts = append(parts, tag)
	}
	return strings.Join(parts, " and ")
}

// withoutEnv removes env entries that match a key prefix.
func withoutEnv(env []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
