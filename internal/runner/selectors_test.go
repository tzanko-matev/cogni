package runner

import (
	"testing"

	"cogni/internal/spec"
)

func TestParseSelectors(t *testing.T) {
	selectors, err := ParseSelectors([]string{"task-a", "task-b@agent-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(selectors) != 2 {
		t.Fatalf("expected 2 selectors, got %d", len(selectors))
	}
	if selectors[0].TaskID != "task-a" || selectors[0].AgentID != "" {
		t.Fatalf("unexpected selector: %+v", selectors[0])
	}
	if selectors[1].TaskID != "task-b" || selectors[1].AgentID != "agent-1" {
		t.Fatalf("unexpected selector: %+v", selectors[1])
	}
}

func TestParseSelectorsErrors(t *testing.T) {
	invalid := []string{"@agent", "task@", "task@@agent"}
	for _, input := range invalid {
		if _, err := ParseSelectors([]string{input}); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestValidateSelectors(t *testing.T) {
	cfg := spec.Config{
		Agents: []spec.AgentConfig{{ID: "agent-1"}},
		Tasks:  []spec.TaskConfig{{ID: "task-1"}},
	}
	if err := ValidateSelectors(cfg, []TaskSelector{{TaskID: "task-1", AgentID: "agent-1"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := ValidateSelectors(cfg, []TaskSelector{{TaskID: "missing"}}); err == nil {
		t.Fatalf("expected task error")
	}
	if err := ValidateSelectors(cfg, []TaskSelector{{TaskID: "task-1", AgentID: "missing"}}); err == nil {
		t.Fatalf("expected agent error")
	}
}
