package runner

import (
	"fmt"
	"strings"

	"cogni/internal/config"
	"cogni/internal/spec"
)

// planTaskRuns resolves tasks, agents, and models into runnable units.
func planTaskRuns(cfg spec.Config, selectors []TaskSelector, agentOverride string) ([]taskRun, error) {
	if err := ValidateSelectors(cfg, selectors); err != nil {
		return nil, err
	}
	agentByID := make(map[string]spec.AgentConfig, len(cfg.Agents))
	for _, agentConfig := range cfg.Agents {
		agentByID[agentConfig.ID] = agentConfig
	}

	selectedAgent := map[string]string{}
	if len(selectors) > 0 {
		for _, selector := range selectors {
			if selector.AgentID != "" {
				selectedAgent[selector.TaskID] = selector.AgentID
			}
		}
	}

	selectedIDs := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		selectedIDs = append(selectedIDs, selector.TaskID)
	}
	orderedTasks, err := config.OrderedTasks(cfg, selectedIDs)
	if err != nil {
		return nil, err
	}

	runs := make([]taskRun, 0, len(orderedTasks))
	for _, task := range orderedTasks {
		agentID := agentOverride
		if agentID == "" {
			if selectorAgent, ok := selectedAgent[task.ID]; ok {
				agentID = selectorAgent
			} else if task.Agent != "" {
				agentID = task.Agent
			} else {
				agentID = cfg.DefaultAgent
			}
		}
		agentConfig, ok := agentByID[agentID]
		if !ok {
			return nil, fmt.Errorf("unknown agent id %q", agentID)
		}
		model := task.Model
		if strings.TrimSpace(model) == "" {
			model = agentConfig.Model
		}
		runs = append(runs, taskRun{
			Task:    task,
			Agent:   agentConfig,
			Model:   model,
			AgentID: agentID,
		})
	}
	return runs, nil
}
