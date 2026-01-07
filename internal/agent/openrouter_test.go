package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cogni/internal/testutil"
)

// TestProviderFromEnvErrors verifies provider environment validation errors.
func TestProviderFromEnvErrors(t *testing.T) {
	t.Setenv("LLM_PROVIDER", "")
	t.Setenv("LLM_API_KEY", "")
	if _, err := ProviderFromEnv("", "model", nil); err == nil {
		t.Fatalf("expected provider error")
	}

	if _, err := ProviderFromEnv("unknown", "model", nil); err == nil {
		t.Fatalf("expected unsupported provider error")
	}

	if _, err := ProviderFromEnv("openrouter", "model", nil); err == nil {
		t.Fatalf("expected missing api key error")
	}
}

// TestOpenRouterStreamParsesMessage verifies streaming message aggregation.
func TestOpenRouterStreamParsesMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello \"}}]}\n\n")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"world\"}}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(server.Close)

	provider, err := NewOpenRouterProvider("model", "key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	ctx := testutil.Context(t, 0)
	stream, err := provider.Stream(ctx, Prompt{
		Instructions: "base",
		InputItems:   []HistoryItem{{Role: "user", Content: HistoryText{Text: "hi"}}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if event.Type != StreamEventMessage || event.Message != "hello world" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if _, err := stream.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

// TestOpenRouterStreamParsesToolCall verifies streaming tool-call parsing.
func TestOpenRouterStreamParsesToolCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		payload := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"query\":\"hi\"}"}}]}}]}`
		fmt.Fprintf(w, "data: %s\n\n", payload)
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	t.Cleanup(server.Close)

	provider, err := NewOpenRouterProvider("model", "key", server.URL, server.Client())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	ctx := testutil.Context(t, 0)
	stream, err := provider.Stream(ctx, Prompt{
		InputItems: []HistoryItem{{Role: "user", Content: HistoryText{Text: "hi"}}},
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	event, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	if event.Type != StreamEventToolCall {
		t.Fatalf("unexpected event type: %v", event.Type)
	}
	if event.ToolCall.Name != "search" || event.ToolCall.ID != "call_1" {
		t.Fatalf("unexpected tool call: %+v", event.ToolCall)
	}
	queryRaw, ok := event.ToolCall.Args["query"]
	if !ok {
		t.Fatalf("missing query arg: %+v", event.ToolCall.Args)
	}
	var query string
	if err := json.Unmarshal(queryRaw, &query); err != nil || query != "hi" {
		t.Fatalf("unexpected tool args: %+v", event.ToolCall.Args)
	}
	if _, err := stream.Recv(); err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

// TestOpenRouterRejectsToolHistoryWithoutID verifies tool history validation.
func TestOpenRouterRejectsToolHistoryWithoutID(t *testing.T) {
	_, err := buildOpenRouterMessages(Prompt{
		InputItems: []HistoryItem{
			{Role: "assistant", Content: ToolCall{Name: "list_files"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "tool call id") {
		t.Fatalf("expected tool call id error, got %v", err)
	}
}
