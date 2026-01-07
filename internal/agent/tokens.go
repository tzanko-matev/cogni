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

func contentString(content HistoryContent) string {
	switch value := content.(type) {
	case HistoryText:
		return value.Text
	case ToolCall:
		args := value.Args
		if args == nil {
			args = ToolCallArgs{}
		}
		raw, _ := json.Marshal(args)
		return value.Name + string(raw)
	case ToolOutput:
		return value.Result.Output
	default:
		return ""
	}
}
