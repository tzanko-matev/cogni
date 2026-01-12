package call

import (
	"context"
	"fmt"
	"io"

	"cogni/internal/agent"
)

// handleResponseStream consumes streamed output and executes any tools.
func handleResponseStream(ctx context.Context, session *agent.Session, stream agent.Stream, executor agent.ToolExecutor, metrics *RunMetrics, opts RunOptions) (bool, error) {
	needsFollowUp := false
	for {
		event, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return needsFollowUp, err
		}
		switch event.Type {
		case agent.StreamEventMessage:
			session.History = append(session.History, agent.HistoryItem{Role: "assistant", Content: agent.HistoryText{Text: event.Message}})
			logVerboseBlock(opts, "LLM output", event.Message, styleHeadingOutput, styleDefault)
		case agent.StreamEventToolCall:
			if event.ToolCall.ID == "" {
				event.ToolCall.ID = fmt.Sprintf("call-%d", len(session.History))
			}
			logVerbose(opts, styleHeadingToolCall, fmt.Sprintf("Tool call id=%s name=%s args=%s", event.ToolCall.ID, event.ToolCall.Name, formatArgs(event.ToolCall.Args)))
			session.History = append(session.History, agent.HistoryItem{Role: "assistant", Content: event.ToolCall})
			result := executor.Execute(ctx, event.ToolCall)
			session.History = append(session.History, agent.HistoryItem{Role: "tool", Content: agent.ToolOutput{
				ToolCallID: event.ToolCall.ID,
				Result:     result,
			}})
			logVerboseToolOutput(opts, fmt.Sprintf("Tool result id=%s name=%s duration=%s bytes=%d truncated=%t error=%s", event.ToolCall.ID, result.Tool, result.Duration, result.OutputBytes, result.Truncated, result.Error), result.Output)
			if metrics != nil {
				if metrics.ToolCalls == nil {
					metrics.ToolCalls = map[string]int{}
				}
				metrics.ToolCalls[event.ToolCall.Name]++
			}
			needsFollowUp = true
		default:
			return needsFollowUp, fmt.Errorf("unknown stream event type: %d", event.Type)
		}
	}
	return needsFollowUp, nil
}
