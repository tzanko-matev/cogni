package agent

import (
	"context"
	"fmt"
	"strings"
)

// TokenCounter estimates token usage for a history slice.
type TokenCounter func(history []HistoryItem) int

// CompactHistory summarizes and trims history when the compaction soft limit is exceeded.
func CompactHistory(ctx context.Context, history []HistoryItem, provider Provider, counter TokenCounter, cfg CompactionConfig) ([]HistoryItem, *CompactionStats, error) {
	cfg = NormalizeCompactionConfig(cfg)
	if counter == nil || cfg.SoftLimit <= 0 {
		return history, nil, nil
	}
	beforeTokens := counter(history)
	if beforeTokens <= cfg.SoftLimit {
		return history, nil, nil
	}
	if provider == nil {
		return history, nil, fmt.Errorf("compaction provider is required")
	}

	filtered := filterSummaryItems(history)
	keep := make([]bool, len(filtered))

	for i, item := range filtered {
		if isDeveloperInstructions(item) || isUserInstructions(item) {
			keep[i] = true
		}
	}
	if envIndex := lastEnvironmentIndex(filtered); envIndex >= 0 {
		keep[envIndex] = true
	}

	userKeep, lastUserIndex := selectRecentUserMessages(filtered, counter, cfg.RecentUserTokenBudget)
	mergeKeep(keep, userKeep)

	toolKeep := selectToolOutputs(filtered, lastUserIndex, cfg.RecentToolOutputLimit)
	mergeKeep(keep, toolKeep)
	mergeKeep(keep, selectToolCallsForOutputs(filtered, toolKeep))

	summaryItems := collectSummaryItems(filtered, keep)
	summary := ""
	if len(summaryItems) > 0 {
		var err error
		summary, err = SummarizeHistory(ctx, provider, summaryItems, cfg.SummaryPrompt, counter, cfg.HardLimit)
		if err != nil {
			return history, nil, err
		}
		summary = strings.TrimSpace(summary)
	}

	compacted := make([]HistoryItem, 0, len(filtered)+1)
	for i, item := range filtered {
		if keep[i] {
			compacted = append(compacted, item)
		}
	}
	if summary != "" {
		compacted = append(compacted, HistoryItem{
			Role:    "assistant",
			Content: HistoryText{Text: SummaryPrefix + summary},
		})
	}

	stats := &CompactionStats{
		BeforeTokens: beforeTokens,
		AfterTokens:  counter(compacted),
	}
	if summary != "" {
		stats.SummaryTokens = countTokens(counter, HistoryItem{
			Role:    "assistant",
			Content: HistoryText{Text: SummaryPrefix + summary},
		})
	}
	return compacted, stats, nil
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

// isSummaryItem reports whether the item is a compaction summary message.
func isSummaryItem(item HistoryItem) bool {
	content, ok := historyText(item)
	if !ok {
		return false
	}
	return strings.HasPrefix(content, SummaryPrefix)
}

// historyText extracts text from a history item when it is a HistoryText payload.
func historyText(item HistoryItem) (string, bool) {
	content, ok := item.Content.(HistoryText)
	if !ok {
		return "", false
	}
	return content.Text, true
}
