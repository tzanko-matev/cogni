package cucumber

import (
	"encoding/json"
	"fmt"
	"strings"
)

type AgentResponse struct {
	ExampleID   string     `json:"example_id"`
	Implemented bool       `json:"implemented"`
	Evidence    []Evidence `json:"evidence,omitempty"`
	Notes       string     `json:"notes,omitempty"`
}

type Evidence struct {
	Path  string `json:"path"`
	Lines []int  `json:"lines"`
}

func ParseAgentResponse(output string) (AgentResponse, error) {
	var response AgentResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return AgentResponse{}, err
	}
	response.ExampleID = strings.TrimSpace(response.ExampleID)
	if response.ExampleID == "" {
		return AgentResponse{}, fmt.Errorf("example_id is required")
	}
	return response, nil
}
