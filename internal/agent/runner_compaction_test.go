package agent_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cogni/internal/agent"
	"cogni/internal/agent/call"
	"cogni/internal/testutil"
)

type capturingProvider struct {
	prompts []agent.Prompt
	streams [][]agent.StreamEvent
}

func (p *capturingProvider) Stream(_ context.Context, prompt agent.Prompt) (agent.Stream, error) {
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
	session := &agent.Session{
		Ctx: agent.TurnContext{
			ModelFamily: agent.ModelFamily{BaseInstructionsTemplate: "base"},
		},
		History: []agent.HistoryItem{
			{Role: "user", Content: agent.HistoryText{Text: "old question"}},
			{Role: "assistant", Content: agent.HistoryText{Text: "old answer"}},
		},
	}
	provider := &capturingProvider{
		streams: [][]agent.StreamEvent{
			{{Type: agent.StreamEventMessage, Message: "summary text"}},
			{{Type: agent.StreamEventMessage, Message: "done"}},
		},
	}
	executor := &fakeExecutor{}

	_, err := call.RunCall(ctx, session, provider, executor, "new question", call.RunOptions{
		TokenCounter: agent.ApproxTokenCount,
		Compaction: agent.CompactionConfig{
			SoftLimit:             1,
			HardLimit:             100,
			RecentUserTokenBudget: 1,
		},
	}, nil)
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	if len(provider.prompts) != 2 {
		t.Fatalf("expected 2 prompts (summary + main), got %d", len(provider.prompts))
	}
	mainPrompt := provider.prompts[1]
	foundSummary := false
	for _, item := range mainPrompt.InputItems {
		text, ok := item.Content.(agent.HistoryText)
		if !ok {
			continue
		}
		if strings.HasPrefix(text.Text, agent.SummaryPrefix) {
			foundSummary = true
			break
		}
	}
	if !foundSummary {
		t.Fatalf("expected summary message in prompt")
	}
}
