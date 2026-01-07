package agent

// HistoryContent represents a single typed content item in a turn.
type HistoryContent interface {
	historyContent()
}

// HistoryText holds plain text content for a history item.
type HistoryText struct {
	Text string
}

func (HistoryText) historyContent() {}

func (ToolCall) historyContent() {}

func (ToolOutput) historyContent() {}
