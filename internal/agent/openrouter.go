package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
)

// defaultOpenRouterBaseURL is the default OpenRouter API base URL.
const defaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"

// OpenRouterProvider implements Provider for the OpenRouter API.
type OpenRouterProvider struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
	Model   string
}

// ProviderFromEnv builds a provider using environment configuration.
func ProviderFromEnv(provider, model string, client *http.Client) (Provider, error) {
	if provider == "" {
		provider = strings.TrimSpace(os.Getenv("LLM_PROVIDER"))
	}
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if provider != "openrouter" {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	if apiKey == "" {
		return nil, fmt.Errorf("LLM_API_KEY is required")
	}
	return NewOpenRouterProvider(model, apiKey, "", client)
}

// NewOpenRouterProvider constructs an OpenRouter provider with explicit settings.
func NewOpenRouterProvider(model, apiKey, baseURL string, client *http.Client) (*OpenRouterProvider, error) {
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("api key is required")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultOpenRouterBaseURL
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &OpenRouterProvider{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
		Model:   model,
	}, nil
}

// Stream sends a prompt to OpenRouter and returns a stream of events.
func (p *OpenRouterProvider) Stream(ctx context.Context, prompt Prompt) (Stream, error) {
	messages, err := buildOpenRouterMessages(prompt)
	if err != nil {
		return nil, err
	}
	requestBody := openRouterRequest{
		Model:    p.Model,
		Stream:   true,
		Messages: messages,
	}
	if len(prompt.Tools) > 0 {
		requestBody.Tools = buildOpenRouterTools(prompt.Tools)
		requestBody.ToolChoice = "auto"
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := p.BaseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter error: %s", strings.TrimSpace(string(body)))
	}

	events, err := parseOpenRouterStream(resp.Body)
	if err != nil {
		return nil, err
	}
	return &staticStream{events: events}, nil
}

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
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
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

// openRouterStreamChunk is a partial SSE payload.
type openRouterStreamChunk struct {
	Choices []openRouterStreamChoice `json:"choices"`
}

// openRouterStreamChoice contains a delta event from OpenRouter.
type openRouterStreamChoice struct {
	Delta        openRouterStreamDelta `json:"delta"`
	FinishReason string                `json:"finish_reason"`
}

// openRouterStreamDelta contains incremental content or tool calls.
type openRouterStreamDelta struct {
	Content   string                     `json:"content"`
	ToolCalls []openRouterStreamToolCall `json:"tool_calls"`
}

// openRouterStreamToolCall represents a streaming tool call delta.
type openRouterStreamToolCall struct {
	Index    int                    `json:"index"`
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openRouterFunctionCall `json:"function"`
}

// toolCallAccumulator gathers streaming tool call fragments.
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// parseOpenRouterStream reads SSE output and converts it into stream events.
func parseOpenRouterStream(reader io.Reader) ([]StreamEvent, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var content strings.Builder
	accumulators := make(map[int]*toolCallAccumulator)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}
		var chunk openRouterStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("parse stream chunk: %w", err)
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				content.WriteString(choice.Delta.Content)
			}
			for _, call := range choice.Delta.ToolCalls {
				acc := accumulators[call.Index]
				if acc == nil {
					acc = &toolCallAccumulator{}
					accumulators[call.Index] = acc
				}
				if call.ID != "" {
					acc.ID = call.ID
				}
				if call.Function.Name != "" {
					acc.Name = call.Function.Name
				}
				if call.Function.Arguments != "" {
					acc.Arguments.WriteString(call.Function.Arguments)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	events := make([]StreamEvent, 0, len(accumulators)+1)
	if content.Len() > 0 {
		events = append(events, StreamEvent{
			Type:    StreamEventMessage,
			Message: content.String(),
		})
	}

	if len(accumulators) > 0 {
		indices := make([]int, 0, len(accumulators))
		for index := range accumulators {
			indices = append(indices, index)
		}
		sort.Ints(indices)
		for _, index := range indices {
			acc := accumulators[index]
			var args ToolCallArgs
			if acc.Arguments.Len() > 0 {
				if err := json.Unmarshal([]byte(acc.Arguments.String()), &args); err != nil {
					return nil, fmt.Errorf("parse tool arguments: %w", err)
				}
			}
			callID := acc.ID
			if callID == "" {
				callID = fmt.Sprintf("call-%d", index)
			}
			events = append(events, StreamEvent{
				Type: StreamEventToolCall,
				ToolCall: ToolCall{
					ID:   callID,
					Name: acc.Name,
					Args: args,
				},
			})
		}
	}

	return events, nil
}

// staticStream exposes a slice of events as a Stream.
type staticStream struct {
	events []StreamEvent
	index  int
}

// Recv returns the next event or io.EOF when complete.
func (s *staticStream) Recv() (StreamEvent, error) {
	if s.index >= len(s.events) {
		return StreamEvent{}, io.EOF
	}
	event := s.events[s.index]
	s.index++
	return event, nil
}
