package agent

import (
	"encoding/json"
	"fmt"
	"strings"
)

// openRouterRequest is the JSON payload sent to OpenRouter.
type openRouterRequest struct {
	Model      string              `json:"model"`
	Stream     bool                `json:"stream"`
	Messages   []openRouterMessage `json:"messages"`
	Tools      []openRouterTool    `json:"tools,omitempty"`
	ToolChoice string              `json:"tool_choice,omitempty"`
}

// openRouterMessage represents a single OpenRouter chat message.
type openRouterMessage struct {
	Role       string               `json:"role"`
	Content    string               `json:"content,omitempty"`
	ToolCalls  []openRouterToolCall `json:"tool_calls,omitempty"`
	ToolCallID string               `json:"tool_call_id,omitempty"`
}

// openRouterTool describes a function tool for OpenRouter.
type openRouterTool struct {
	Type     string                       `json:"type"`
	Function openRouterFunctionDefinition `json:"function"`
}

// openRouterFunctionDefinition describes a tool's function signature.
type openRouterFunctionDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Parameters  *ToolSchema `json:"parameters,omitempty"`
}

// openRouterToolCall represents a tool call emitted by OpenRouter.
type openRouterToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openRouterFunctionCall `json:"function"`
}

// openRouterFunctionCall describes the name and arguments of a tool call.
type openRouterFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// buildOpenRouterMessages converts a prompt into OpenRouter message payloads.
func buildOpenRouterMessages(prompt Prompt) ([]openRouterMessage, error) {
	messages := make([]openRouterMessage, 0, len(prompt.InputItems)+1)
	if strings.TrimSpace(prompt.Instructions) != "" {
		messages = append(messages, openRouterMessage{
			Role:    "system",
			Content: prompt.Instructions,
		})
	}
	for _, item := range prompt.InputItems {
		msg, err := toOpenRouterMessage(item)
		if err != nil {
			return nil, err
		}
		if msg.Role != "" {
			messages = append(messages, msg)
		}
	}
	return messages, nil
}

// toOpenRouterMessage converts a history item into an OpenRouter message.
func toOpenRouterMessage(item HistoryItem) (openRouterMessage, error) {
	role := item.Role
	if role == "developer" {
		role = "system"
	}
	switch content := item.Content.(type) {
	case HistoryText:
		return openRouterMessage{Role: role, Content: content.Text}, nil
	case ToolCall:
		args := content.Args
		if args == nil {
			args = ToolCallArgs{}
		}
		payload, err := json.Marshal(args)
		if err != nil {
			return openRouterMessage{}, fmt.Errorf("marshal tool args: %w", err)
		}
		if content.ID == "" {
			return openRouterMessage{}, fmt.Errorf("tool call id is required")
		}
		return openRouterMessage{
			Role: role,
			ToolCalls: []openRouterToolCall{{
				ID:   content.ID,
				Type: "function",
				Function: openRouterFunctionCall{
					Name:      content.Name,
					Arguments: string(payload),
				},
			}},
		}, nil
	case ToolOutput:
		return openRouterMessage{
			Role:       "tool",
			Content:    content.Result.Output,
			ToolCallID: content.ToolCallID,
		}, nil
	default:
		return openRouterMessage{}, fmt.Errorf("unsupported history content type")
	}
}

// buildOpenRouterTools converts tool definitions into OpenRouter tool payloads.
func buildOpenRouterTools(defs []ToolDefinition) []openRouterTool {
	tools := make([]openRouterTool, 0, len(defs))
	for _, def := range defs {
		params := def.Parameters
		if params == nil {
			defaultSchema := ToolSchema{Type: "object"}
			params = &defaultSchema
		}
		tools = append(tools, openRouterTool{
			Type: "function",
			Function: openRouterFunctionDefinition{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  params,
			},
		})
	}
	return tools
}
