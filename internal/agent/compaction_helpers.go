package agent

// filterSummaryItems removes prior summary messages from history.
func filterSummaryItems(history []HistoryItem) []HistoryItem {
	filtered := make([]HistoryItem, 0, len(history))
	for _, item := range history {
		if isSummaryItem(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

// lastEnvironmentIndex finds the most recent environment context or diff item.
func lastEnvironmentIndex(history []HistoryItem) int {
	for i := len(history) - 1; i >= 0; i-- {
		if isEnvironmentItem(history[i]) {
			return i
		}
	}
	return -1
}

// selectRecentUserMessages keeps recent user messages within a token budget.
func selectRecentUserMessages(history []HistoryItem, counter TokenCounter, budget int) ([]bool, int) {
	keep := make([]bool, len(history))
	total := 0
	lastUserIndex := -1
	for i := len(history) - 1; i >= 0; i-- {
		if !isUserMessage(history[i]) {
			continue
		}
		tokens := countTokens(counter, history[i])
		if budget > 0 && total+tokens > budget && total > 0 {
			break
		}
		keep[i] = true
		total += tokens
		if lastUserIndex == -1 {
			lastUserIndex = i
		}
	}
	if lastUserIndex == -1 {
		for i := len(history) - 1; i >= 0; i-- {
			if isUserMessage(history[i]) {
				keep[i] = true
				lastUserIndex = i
				break
			}
		}
	}
	return keep, lastUserIndex
}

// selectToolOutputs retains tool outputs based on policy.
func selectToolOutputs(history []HistoryItem, lastUserIndex int, limit int) []bool {
	keep := make([]bool, len(history))
	if limit > 0 {
		count := 0
		for i := len(history) - 1; i >= 0 && count < limit; i-- {
			if history[i].Role == "tool" {
				keep[i] = true
				count++
			}
		}
		return keep
	}
	if lastUserIndex < 0 {
		return keep
	}
	for i := lastUserIndex + 1; i < len(history); i++ {
		if history[i].Role == "tool" {
			keep[i] = true
		}
	}
	return keep
}

// selectToolCallsForOutputs keeps tool call inputs paired with retained outputs.
func selectToolCallsForOutputs(history []HistoryItem, outputKeep []bool) []bool {
	keep := make([]bool, len(history))
	for i, keepOutput := range outputKeep {
		if !keepOutput {
			continue
		}
		output, ok := history[i].Content.(ToolOutput)
		if !ok {
			continue
		}
		callIndex := findToolCallIndex(history, i, output.ToolCallID)
		if callIndex >= 0 {
			keep[callIndex] = true
		}
	}
	return keep
}

// findToolCallIndex locates the tool call that produced a tool output.
func findToolCallIndex(history []HistoryItem, outputIndex int, callID string) int {
	if callID == "" {
		return -1
	}
	for i := outputIndex - 1; i >= 0; i-- {
		call, ok := history[i].Content.(ToolCall)
		if !ok {
			continue
		}
		if call.ID == callID {
			return i
		}
	}
	return -1
}

// collectSummaryItems returns history items that were removed for summarization.
func collectSummaryItems(history []HistoryItem, keep []bool) []HistoryItem {
	items := make([]HistoryItem, 0, len(history))
	for i, item := range history {
		if keep[i] {
			continue
		}
		items = append(items, item)
	}
	return items
}

// mergeKeep merges keep flags into the target slice.
func mergeKeep(target []bool, source []bool) {
	for i := range target {
		if i < len(source) && source[i] {
			target[i] = true
		}
	}
}

// countTokens estimates token usage for a single history item.
func countTokens(counter TokenCounter, item HistoryItem) int {
	if counter == nil {
		return ApproxTokenCount([]HistoryItem{item})
	}
	return counter([]HistoryItem{item})
}
