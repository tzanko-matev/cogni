package question

import "strings"

// NormalizeAnswerText trims whitespace and lowercases an answer for matching.
func NormalizeAnswerText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
