package agent

import (
	"encoding/json"
)

// ApproxTokenCount estimates token usage by dividing character count by four.
func ApproxTokenCount(history []HistoryItem) int {
	total := 0
	for _, item := range history {
		total += len(contentString(item.Content))
	}
	return total / 4
}

func contentString(content any) string {
	switch value := content.(type) {
	case string:
		return value
	case ToolCall:
		raw, _ := json.Marshal(value.Args)
		return value.Name + string(raw)
	case ToolOutput:
		return value.Result.Output
	default:
		return ""
	}
}
