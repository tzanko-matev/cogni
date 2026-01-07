package config

import (
	"fmt"
	"strings"

	"cogni/internal/spec"
)

// validateAgents checks agent entries and returns a map of agent IDs.
func validateAgents(cfg *spec.Config, add issueAdder) map[string]struct{} {
	agentIDs := map[string]struct{}{}
	if len(cfg.Agents) == 0 {
		add("agents", "at least one agent is required")
	}
	for i, agent := range cfg.Agents {
		fieldPrefix := fmt.Sprintf("agents[%d]", i)
		id := strings.TrimSpace(agent.ID)
		if id == "" {
			add(fieldPrefix+".id", "is required")
		} else if _, exists := agentIDs[id]; exists {
			add("agents.id", fmt.Sprintf("duplicate id %q", id))
		} else {
			agentIDs[id] = struct{}{}
		}
		if agent.Type == "" {
			add(fieldPrefix+".type", "is required")
		} else if agent.Type != "builtin" {
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", agent.Type))
		}
		if strings.TrimSpace(agent.Provider) == "" {
			add(fieldPrefix+".provider", "is required")
		}
		if strings.TrimSpace(agent.Model) == "" {
			add(fieldPrefix+".model", "is required")
		}
		if agent.MaxSteps < 0 {
			add(fieldPrefix+".max_steps", "must be >= 0")
		}
	}
	return agentIDs
}

// validateDefaultAgent ensures the configured default agent exists.
func validateDefaultAgent(cfg *spec.Config, agentIDs map[string]struct{}, add issueAdder) {
	defaultAgent := strings.TrimSpace(cfg.DefaultAgent)
	if defaultAgent == "" {
		add("default_agent", "is required")
		return
	}
	if _, ok := agentIDs[defaultAgent]; !ok {
		add("default_agent", fmt.Sprintf("unknown agent %q", defaultAgent))
	}
}
