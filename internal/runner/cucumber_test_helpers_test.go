package runner

import (
	"context"
	"encoding/json"
	"strings"

	"cogni/internal/agent"
)

// cucumberProvider fakes agent responses for cucumber eval tests.
type cucumberProvider struct {
	implementedByID map[string]bool
	responseIDs     []string
}

// cucumberBatchResult is a single response entry returned by cucumberProvider.
type cucumberBatchResult struct {
	ExampleID   string `json:"example_id"`
	Implemented bool   `json:"implemented"`
}

// cucumberBatchResponse is the aggregated response payload for a batch.
type cucumberBatchResponse struct {
	Results []cucumberBatchResult `json:"results"`
}

// Stream returns a canned batch response based on the prompt contents.
func (p cucumberProvider) Stream(_ context.Context, prompt agent.Prompt) (agent.Stream, error) {
	exampleIDs := p.responseIDs
	if len(exampleIDs) == 0 {
		exampleIDs = extractExampleIDs(prompt)
	}
	results := make([]cucumberBatchResult, 0, len(exampleIDs))
	for _, exampleID := range exampleIDs {
		results = append(results, cucumberBatchResult{
			ExampleID:   exampleID,
			Implemented: p.implementedByID[exampleID],
		})
	}
	payload, err := json.Marshal(cucumberBatchResponse{Results: results})
	if err != nil {
		return nil, err
	}
	message := string(payload)
	return &fakeStream{events: []agent.StreamEvent{{Type: agent.StreamEventMessage, Message: message}}}, nil
}

// extractExampleIDs extracts example IDs from a cucumber eval prompt.
func extractExampleIDs(prompt agent.Prompt) []string {
	for i := len(prompt.InputItems) - 1; i >= 0; i-- {
		item := prompt.InputItems[i]
		if item.Role != "user" {
			continue
		}
		text, ok := item.Content.(agent.HistoryText)
		if !ok {
			continue
		}
		if ids := parseExampleIDsFromHeader(text.Text); len(ids) > 0 {
			return ids
		}
		if ids := parseExampleIDsFromMarker(text.Text); len(ids) > 0 {
			return ids
		}
	}
	return nil
}

// parseExampleIDsFromHeader collects IDs listed after the built-in prompt header.
func parseExampleIDsFromHeader(promptText string) []string {
	const header = "Expected Example IDs (one per line):"
	lines := strings.Split(promptText, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != header {
			continue
		}
		return collectExampleIDs(lines[i+1:])
	}
	return nil
}

// parseExampleIDsFromMarker collects IDs listed after the legacy prompt marker.
func parseExampleIDsFromMarker(promptText string) []string {
	if !strings.Contains(promptText, "example_ids:") {
		return nil
	}
	parts := strings.SplitN(promptText, "example_ids:", 2)
	if len(parts) != 2 {
		return nil
	}
	return collectExampleIDs(strings.Split(parts[1], "\n"))
}

// collectExampleIDs trims IDs until a blank or footer line.
func collectExampleIDs(lines []string) []string {
	ids := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "For each example ID") || strings.HasPrefix(trimmed, "Return ONLY JSON") {
			break
		}
		ids = append(ids, trimmed)
	}
	return ids
}
