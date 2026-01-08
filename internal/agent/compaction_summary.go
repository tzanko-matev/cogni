package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// SummarizeHistory generates a summary for the provided history items.
func SummarizeHistory(ctx context.Context, provider Provider, items []HistoryItem, prompt string, counter TokenCounter, limit int) (string, error) {
	if provider == nil {
		return "", fmt.Errorf("summary provider is required")
	}
	summaryPrompt := strings.TrimSpace(prompt)
	if summaryPrompt == "" {
		summaryPrompt = DefaultSummaryPrompt
	}
	trimmed := trimSummaryItems(items, summaryPrompt, counter, limit)
	if len(trimmed) == 0 {
		return "", nil
	}
	if limit > 0 && estimatePromptTokens(counter, summaryPrompt, trimmed) > limit {
		return "", fmt.Errorf("summary prompt exceeds limit after trimming")
	}

	stream, err := provider.Stream(ctx, Prompt{
		Instructions: summaryPrompt,
		InputItems:   trimmed,
	})
	if err != nil {
		return "", err
	}

	var summary strings.Builder
	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		switch event.Type {
		case StreamEventMessage:
			summary.WriteString(event.Message)
		case StreamEventToolCall:
			return "", fmt.Errorf("summary stream emitted tool call")
		default:
			return "", fmt.Errorf("summary stream emitted unknown event type %d", event.Type)
		}
	}
	return strings.TrimSpace(summary.String()), nil
}

// trimSummaryItems removes oldest items until the prompt fits within limits.
func trimSummaryItems(items []HistoryItem, prompt string, counter TokenCounter, limit int) []HistoryItem {
	if limit <= 0 {
		return items
	}
	trimmed := items
	for len(trimmed) > 0 && estimatePromptTokens(counter, prompt, trimmed) > limit {
		trimmed = trimmed[1:]
	}
	return trimmed
}

// estimatePromptTokens approximates prompt token usage including instructions.
func estimatePromptTokens(counter TokenCounter, prompt string, items []HistoryItem) int {
	tokens := 0
	if counter != nil {
		tokens = counter(items)
	} else {
		tokens = ApproxTokenCount(items)
	}
	if strings.TrimSpace(prompt) != "" {
		tokens += len(prompt) / 4
	}
	return tokens
}
