package runner

import (
	"fmt"
	"strings"

	"cogni/internal/spec"
)

type TaskSelector struct {
	TaskID  string
	AgentID string
}

func ParseSelectors(inputs []string) ([]TaskSelector, error) {
	selectors := make([]TaskSelector, 0, len(inputs))
	for _, input := range inputs {
		trimmed := strings.TrimSpace(input)
		if trimmed == "" {
			continue
		}
		if strings.Count(trimmed, "@") > 1 {
			return nil, fmt.Errorf("invalid selector %q", input)
		}
		parts := strings.SplitN(trimmed, "@", 2)
		taskID := strings.TrimSpace(parts[0])
		if taskID == "" {
			return nil, fmt.Errorf("invalid selector %q", input)
		}
		selector := TaskSelector{TaskID: taskID}
		if len(parts) == 2 {
			agentID := strings.TrimSpace(parts[1])
			if agentID == "" {
				return nil, fmt.Errorf("invalid selector %q", input)
			}
			selector.AgentID = agentID
		}
		selectors = append(selectors, selector)
	}
	return selectors, nil
}

func ValidateSelectors(cfg spec.Config, selectors []TaskSelector) error {
	if len(selectors) == 0 {
		return nil
	}
	taskIDs := make(map[string]struct{}, len(cfg.Tasks))
	for _, task := range cfg.Tasks {
		taskIDs[task.ID] = struct{}{}
	}
	agentIDs := make(map[string]struct{}, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		agentIDs[agent.ID] = struct{}{}
	}
	for _, selector := range selectors {
		if _, ok := taskIDs[selector.TaskID]; !ok {
			return fmt.Errorf("unknown task id %q", selector.TaskID)
		}
		if selector.AgentID != "" {
			if _, ok := agentIDs[selector.AgentID]; !ok {
				return fmt.Errorf("unknown agent id %q", selector.AgentID)
			}
		}
	}
	return nil
}
