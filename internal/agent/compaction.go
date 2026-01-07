package agent

import (
	"strings"
)

// TokenCounter estimates token usage for a history slice.
type TokenCounter func(history []HistoryItem) int

// CompactHistory trims history while keeping instructions, tool outputs, and the latest user/env messages.
func CompactHistory(history []HistoryItem, counter TokenCounter, limit int) []HistoryItem {
	if counter == nil || limit <= 0 {
		return history
	}
	if counter(history) <= limit {
		return history
	}

	keep := make([]bool, len(history))
	lastEnv := -1
	lastUser := -1
	for i := len(history) - 1; i >= 0; i-- {
		item := history[i]
		if lastEnv == -1 && isEnvironmentItem(item) {
			lastEnv = i
		}
		if lastUser == -1 && isUserMessage(item) {
			lastUser = i
		}
	}

	if lastEnv >= 0 {
		keep[lastEnv] = true
	}
	if lastUser >= 0 {
		keep[lastUser] = true
	}

	for i, item := range history {
		if isDeveloperInstructions(item) || isUserInstructions(item) || item.Role == "tool" {
			keep[i] = true
		}
	}

	compacted := make([]HistoryItem, 0, len(history))
	for i, item := range history {
		if keep[i] {
			compacted = append(compacted, item)
		}
	}
	return compacted
}

// isDeveloperInstructions reports whether the item holds developer guidance.
func isDeveloperInstructions(item HistoryItem) bool {
	return item.Role == "developer"
}

// isUserInstructions reports whether the item holds user guidance.
func isUserInstructions(item HistoryItem) bool {
	if item.Role != "user" {
		return false
	}
	content, ok := historyText(item)
	if !ok {
		return false
	}
	return strings.HasPrefix(content, "# AGENTS.md instructions for ")
}

// isEnvironmentItem reports whether the item holds environment context or diff data.
func isEnvironmentItem(item HistoryItem) bool {
	if item.Role != "user" {
		return false
	}
	content, ok := historyText(item)
	if !ok {
		return false
	}
	return strings.HasPrefix(content, "<environment_context>") ||
		strings.HasPrefix(content, "<environment_diff>")
}

// isUserMessage reports whether the item is a normal user message.
func isUserMessage(item HistoryItem) bool {
	if item.Role != "user" {
		return false
	}
	if isUserInstructions(item) || isEnvironmentItem(item) {
		return false
	}
	return true
}

// historyText extracts text from a history item when it is a HistoryText payload.
func historyText(item HistoryItem) (string, bool) {
	content, ok := item.Content.(HistoryText)
	if !ok {
		return "", false
	}
	return content.Text, true
}
