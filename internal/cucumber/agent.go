package cucumber

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// AgentResponse describes one example verdict from the agent.
type AgentResponse struct {
	ExampleID   string     `json:"example_id"`
	Implemented bool       `json:"implemented"`
	Evidence    []Evidence `json:"evidence,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

// AgentBatchResponse wraps a set of agent responses.
type AgentBatchResponse struct {
	Results []AgentResponse `json:"results"`
}

// Evidence captures a file reference used by the agent.
type Evidence struct {
	Path  string `json:"path"`
	Lines []int  `json:"lines"`
}

// BatchValidationError captures missing, extra, and duplicate example IDs.
type BatchValidationError struct {
	Missing   []string
	Extra     []string
	Duplicate []string
}

// Error formats batch validation errors for display.
func (err BatchValidationError) Error() string {
	parts := make([]string, 0, 3)
	if len(err.Missing) > 0 {
		parts = append(parts, fmt.Sprintf("missing=%s", strings.Join(err.Missing, ",")))
	}
	if len(err.Extra) > 0 {
		parts = append(parts, fmt.Sprintf("extra=%s", strings.Join(err.Extra, ",")))
	}
	if len(err.Duplicate) > 0 {
		parts = append(parts, fmt.Sprintf("duplicate=%s", strings.Join(err.Duplicate, ",")))
	}
	return fmt.Sprintf("invalid example_id set (%s)", strings.Join(parts, " "))
}

// ParseAgentBatchResponse parses a JSON batch response from an agent.
func ParseAgentBatchResponse(output string) (AgentBatchResponse, error) {
	var response AgentBatchResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return AgentBatchResponse{}, err
	}
	return response, nil
}

// ValidateAgentBatchResponse validates agent responses against expected IDs.
func ValidateAgentBatchResponse(expectedIDs []string, response AgentBatchResponse) (map[string]AgentResponse, error) {
	expected := make(map[string]struct{}, len(expectedIDs))
	for _, id := range expectedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		expected[id] = struct{}{}
	}

	results := make(map[string]AgentResponse, len(response.Results))
	var extra []string
	var duplicate []string
	for _, result := range response.Results {
		id := strings.TrimSpace(result.ExampleID)
		if id == "" {
			return results, fmt.Errorf("example_id is required")
		}
		if _, exists := results[id]; exists {
			duplicate = append(duplicate, id)
			continue
		}
		result.ExampleID = id
		results[id] = result
		if _, ok := expected[id]; !ok {
			extra = append(extra, id)
		}
	}

	var missing []string
	for _, id := range expectedIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := results[id]; !ok {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 || len(extra) > 0 || len(duplicate) > 0 {
		sort.Strings(missing)
		sort.Strings(extra)
		sort.Strings(duplicate)
		return results, BatchValidationError{Missing: missing, Extra: extra, Duplicate: duplicate}
	}

	return results, nil
}
