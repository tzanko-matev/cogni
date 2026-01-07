package agent

// HistoryContent represents a single typed content item in a turn.
type HistoryContent interface {
	historyContent()
}

// HistoryText holds plain text content for a history item.
type HistoryText struct {
	Text string
}

// historyContent marks HistoryText as HistoryContent.
func (HistoryText) historyContent() {}

// historyContent marks ToolCall as HistoryContent.
func (ToolCall) historyContent() {}

// historyContent marks ToolOutput as HistoryContent.
func (ToolOutput) historyContent() {}
