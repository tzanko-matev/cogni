package agent

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

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
