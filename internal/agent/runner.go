package agent

import (
	"context"
	"fmt"
	"io"

	"cogni/internal/tools"
)

type StreamEventType int

const (
	StreamEventMessage StreamEventType = iota
	StreamEventToolCall
)

type StreamEvent struct {
	Type     StreamEventType
	Message  string
	ToolCall ToolCall
}

type Stream interface {
	Recv() (StreamEvent, error)
}

type Provider interface {
	Stream(ctx context.Context, prompt Prompt) (Stream, error)
}

type ToolCall struct {
	Name string
	Args map[string]any
}

type ToolExecutor interface {
	Execute(ctx context.Context, call ToolCall) tools.CallResult
}

func RunTurn(ctx context.Context, session *Session, provider Provider, executor ToolExecutor, userText string, counter TokenCounter, limit int) error {
	session.History = append(session.History, HistoryItem{Role: "user", Content: userText})
	if counter != nil && limit > 0 && counter(session.History) > limit {
		session.History = CompactHistory(session.History, counter, limit)
	}
	for {
		prompt := BuildPrompt(session.Ctx, session.History)
		stream, err := provider.Stream(ctx, prompt)
		if err != nil {
			return err
		}
		needsFollowUp, err := HandleResponseStream(ctx, session, stream, executor)
		if err != nil {
			return err
		}
		if !needsFollowUp {
			return nil
		}
	}
}

func HandleResponseStream(ctx context.Context, session *Session, stream Stream, executor ToolExecutor) (bool, error) {
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
		case StreamEventMessage:
			session.History = append(session.History, HistoryItem{Role: "assistant", Content: event.Message})
		case StreamEventToolCall:
			session.History = append(session.History, HistoryItem{Role: "assistant", Content: event.ToolCall})
			result := executor.Execute(ctx, event.ToolCall)
			session.History = append(session.History, HistoryItem{Role: "tool", Content: result})
			needsFollowUp = true
		default:
			return needsFollowUp, fmt.Errorf("unknown stream event type: %d", event.Type)
		}
	}
	return needsFollowUp, nil
}
