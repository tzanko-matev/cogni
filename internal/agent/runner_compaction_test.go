package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

type capturingProvider struct {
	prompts []Prompt
	streams [][]StreamEvent
}

func (p *capturingProvider) Stream(_ context.Context, prompt Prompt) (Stream, error) {
	p.prompts = append(p.prompts, prompt)
	if len(p.streams) == 0 {
		return nil, fmt.Errorf("no streams configured")
	}
	events := p.streams[0]
	p.streams = p.streams[1:]
	return &fakeStream{events: events}, nil
}

// TestRunTurnCompactionInsertsSummary verifies summaries are inserted into prompts.
func TestRunTurnCompactionInsertsSummary(t *testing.T) {
	ctx := testutil.Context(t, 0)
	session := &Session{
		Ctx: TurnContext{
			ModelFamily: ModelFamily{BaseInstructionsTemplate: "base"},
		},
		History: []HistoryItem{
			{Role: "user", Content: HistoryText{Text: "old question"}},
			{Role: "assistant", Content: HistoryText{Text: "old answer"}},
		},
	}
	provider := &capturingProvider{
		streams: [][]StreamEvent{
			{{Type: StreamEventMessage, Message: "summary text"}},
			{{Type: StreamEventMessage, Message: "done"}},
		},
	}
	executor := &fakeExecutor{}

	_, err := RunTurn(ctx, session, provider, executor, "new question", RunOptions{
		TokenCounter: ApproxTokenCount,
		Compaction: CompactionConfig{
			SoftLimit:             1,
			HardLimit:             100,
			RecentUserTokenBudget: 1,
		},
	})
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if len(provider.prompts) != 2 {
		t.Fatalf("expected 2 prompts (summary + main), got %d", len(provider.prompts))
	}
	mainPrompt := provider.prompts[1]
	foundSummary := false
	for _, item := range mainPrompt.InputItems {
		text, ok := item.Content.(HistoryText)
		if !ok {
			continue
		}
		if strings.HasPrefix(text.Text, SummaryPrefix) {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Fatalf("expected summary message in prompt")
	}
}
