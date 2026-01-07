package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// formatPrompt formats prompt contents for verbose logging.
func formatPrompt(prompt Prompt) string {
	var builder strings.Builder
	if strings.TrimSpace(prompt.Instructions) != "" {
		builder.WriteString("instructions:\n")
		builder.WriteString(prompt.Instructions)
		builder.WriteString("\n")
	}
	if len(prompt.Tools) > 0 {
		toolNames := make([]string, 0, len(prompt.Tools))
		for _, tool := range prompt.Tools {
			toolNames = append(toolNames, tool.Name)
		}
		builder.WriteString("tools: ")
		builder.WriteString(strings.Join(toolNames, ", "))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(prompt.OutputSchema) != "" {
		builder.WriteString("output_schema:\n")
		builder.WriteString(prompt.OutputSchema)
		builder.WriteString("\n")
	}
	if len(prompt.InputItems) > 0 {
		builder.WriteString("input_items:\n")
		for _, item := range prompt.InputItems {
			builder.WriteString(formatHistoryItem(item))
		}
	}
	return strings.TrimRight(builder.String(), "\n")
}

// formatHistoryItem renders a single history item for verbose output.
func formatHistoryItem(item HistoryItem) string {
	switch content := item.Content.(type) {
	case HistoryText:
		return fmt.Sprintf("- %s: %s\n", item.Role, content.Text)
	case ToolCall:
		return fmt.Sprintf("- %s: tool_call id=%s name=%s args=%s\n", item.Role, content.ID, content.Name, formatArgs(content.Args))
	case ToolOutput:
		output := strings.TrimRight(content.Result.Output, "\n")
		if output != "" {
			output = truncateVerboseInline(limitOutputLines(output, verboseToolOutputMaxLines))
			output = indentLines(output, "  ")
			return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error, output)
		}
		return fmt.Sprintf("- %s: tool_output call_id=%s tool=%s bytes=%d truncated=%t error=%s\n", item.Role, content.ToolCallID, content.Result.Tool, content.Result.OutputBytes, content.Result.Truncated, content.Result.Error)
	default:
		return fmt.Sprintf("- %s: %v\n", item.Role, content)
	}
}

// formatArgs renders tool call arguments for verbose logging.
func formatArgs(args ToolCallArgs) string {
	if len(args) == 0 {
		return "{}"
	}
	payload, err := json.Marshal(args)
	if err != nil {
		return "<invalid args>"
	}
	return string(payload)
}

// indentLines prefixes each line in a string with a prefix.
func indentLines(value, prefix string) string {
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
