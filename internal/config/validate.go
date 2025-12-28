package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/spec"
)

type Issue struct {
	Field   string
	Message string
}

type ValidationError struct {
	Issues []Issue
}

func (err *ValidationError) Error() string {
	if err == nil || len(err.Issues) == 0 {
		return "config validation failed"
	}
	lines := make([]string, 0, len(err.Issues))
	for _, issue := range err.Issues {
		lines = append(lines, fmt.Sprintf("%s: %s", issue.Field, issue.Message))
	}
	return strings.Join(lines, "\n")
}

func Validate(cfg *spec.Config, baseDir string) error {
	var issues []Issue
	add := func(field, message string) {
		issues = append(issues, Issue{Field: field, Message: message})
	}

	if cfg.Version == 0 {
		add("version", "is required")
	} else if cfg.Version != 1 {
		add("version", fmt.Sprintf("unsupported version %d", cfg.Version))
	}

	if strings.TrimSpace(cfg.Repo.OutputDir) == "" {
		add("repo.output_dir", "is required")
	}

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

	defaultAgent := strings.TrimSpace(cfg.DefaultAgent)
	if defaultAgent == "" {
		add("default_agent", "is required")
	} else if _, ok := agentIDs[defaultAgent]; !ok {
		add("default_agent", fmt.Sprintf("unknown agent %q", defaultAgent))
	}

	if baseDir == "" {
		baseDir = "."
	}

	taskIDs := map[string]struct{}{}
	for i, task := range cfg.Tasks {
		fieldPrefix := fmt.Sprintf("tasks[%d]", i)
		id := strings.TrimSpace(task.ID)
		if id == "" {
			add(fieldPrefix+".id", "is required")
		} else if _, exists := taskIDs[id]; exists {
			add("tasks.id", fmt.Sprintf("duplicate id %q", id))
		} else {
			taskIDs[id] = struct{}{}
		}
		if strings.TrimSpace(task.Type) == "" {
			add(fieldPrefix+".type", "is required")
		} else if task.Type != "qa" {
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", task.Type))
		}
		if strings.TrimSpace(task.Prompt) == "" {
			add(fieldPrefix+".prompt", "is required")
		}
		if strings.TrimSpace(task.Agent) == "" {
			add(fieldPrefix+".agent", "is required")
		} else if _, ok := agentIDs[task.Agent]; !ok {
			add(fieldPrefix+".agent", fmt.Sprintf("unknown agent %q", task.Agent))
		}
		if task.Budget.MaxTokens < 0 {
			add(fieldPrefix+".budget.max_tokens", "must be >= 0")
		}
		if task.Budget.MaxSeconds < 0 {
			add(fieldPrefix+".budget.max_seconds", "must be >= 0")
		}
		if task.Budget.MaxSteps < 0 {
			add(fieldPrefix+".budget.max_steps", "must be >= 0")
		}
		if strings.TrimSpace(task.Eval.JSONSchema) != "" {
			schemaPath := task.Eval.JSONSchema
			if !filepath.IsAbs(schemaPath) {
				schemaPath = filepath.Join(baseDir, schemaPath)
			}
			info, err := os.Stat(schemaPath)
			if err != nil {
				add(fieldPrefix+".eval.json_schema", fmt.Sprintf("schema not found at %q", task.Eval.JSONSchema))
			} else if info.IsDir() {
				add(fieldPrefix+".eval.json_schema", fmt.Sprintf("schema path %q is a directory", task.Eval.JSONSchema))
			}
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}
