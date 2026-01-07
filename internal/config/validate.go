package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cogni/internal/spec"
)

// Issue captures a validation problem with a config field.
type Issue struct {
	Field   string
	Message string
}

// ValidationError aggregates config validation issues.
type ValidationError struct {
	Issues []Issue
}

// Error renders validation errors as a multi-line string.
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

// Validate checks a config for correctness and referenced files.
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

	adapterIDs := map[string]struct{}{}
	for i, adapter := range cfg.Adapters {
		fieldPrefix := fmt.Sprintf("adapters[%d]", i)
		id := strings.TrimSpace(adapter.ID)
		if id == "" {
			add(fieldPrefix+".id", "is required")
		} else if _, exists := adapterIDs[id]; exists {
			add("adapters.id", fmt.Sprintf("duplicate id %q", id))
		} else {
			adapterIDs[id] = struct{}{}
		}

		adapterType := strings.TrimSpace(adapter.Type)
		switch adapterType {
		case "cucumber":
			if strings.TrimSpace(adapter.Runner) == "" {
				add(fieldPrefix+".runner", "is required")
			} else if adapter.Runner != "godog" {
				add(fieldPrefix+".runner", fmt.Sprintf("unsupported runner %q", adapter.Runner))
			}
			if strings.TrimSpace(adapter.Formatter) == "" {
				add(fieldPrefix+".formatter", "is required")
			} else if adapter.Formatter != "json" {
				add(fieldPrefix+".formatter", fmt.Sprintf("unsupported formatter %q", adapter.Formatter))
			}
		case "cucumber_manual":
			if strings.TrimSpace(adapter.ExpectationsDir) == "" {
				add(fieldPrefix+".expectations_dir", "is required")
			}
		case "":
			add(fieldPrefix+".type", "is required")
		default:
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", adapter.Type))
		}

		if len(adapter.FeatureRoots) == 0 {
			add(fieldPrefix+".feature_roots", "must include at least one entry")
		}
		for rootIndex, root := range adapter.FeatureRoots {
			root = strings.TrimSpace(root)
			if root == "" {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), "is required")
				continue
			}
			rootPath := root
			if !filepath.IsAbs(rootPath) {
				rootPath = filepath.Join(baseDir, rootPath)
			}
			info, err := os.Stat(rootPath)
			if err != nil {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), fmt.Sprintf("path not found at %q", root))
				continue
			}
			if !info.IsDir() {
				add(fmt.Sprintf("%s.feature_roots[%d]", fieldPrefix, rootIndex), fmt.Sprintf("path %q is not a directory", root))
			}
		}
		if adapterType == "cucumber_manual" && strings.TrimSpace(adapter.ExpectationsDir) != "" {
			dirPath := adapter.ExpectationsDir
			if !filepath.IsAbs(dirPath) {
				dirPath = filepath.Join(baseDir, dirPath)
			}
			info, err := os.Stat(dirPath)
			if err != nil {
				add(fieldPrefix+".expectations_dir", fmt.Sprintf("path not found at %q", adapter.ExpectationsDir))
			} else if !info.IsDir() {
				add(fieldPrefix+".expectations_dir", fmt.Sprintf("path %q is not a directory", adapter.ExpectationsDir))
			}
		}
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
		taskType := strings.TrimSpace(task.Type)
		if taskType == "" {
			add(fieldPrefix+".type", "is required")
		} else if taskType != "qa" && taskType != "cucumber_eval" {
			add(fieldPrefix+".type", fmt.Sprintf("unsupported type %q", task.Type))
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
		switch taskType {
		case "qa":
			if strings.TrimSpace(task.Prompt) == "" {
				add(fieldPrefix+".prompt", "is required")
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
		case "cucumber_eval":
			if strings.TrimSpace(task.PromptTemplate) == "" {
				add(fieldPrefix+".prompt_template", "is required")
			}
			adapterID := strings.TrimSpace(task.Adapter)
			if adapterID == "" {
				add(fieldPrefix+".adapter", "is required")
			} else if _, ok := adapterIDs[adapterID]; !ok {
				add(fieldPrefix+".adapter", fmt.Sprintf("unknown adapter %q", adapterID))
			}
			if len(task.Features) == 0 {
				add(fieldPrefix+".features", "must include at least one entry")
			}
			for featureIndex, feature := range task.Features {
				feature = strings.TrimSpace(feature)
				if feature == "" {
					add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), "is required")
					continue
				}
				featurePath := feature
				if !filepath.IsAbs(featurePath) {
					featurePath = filepath.Join(baseDir, featurePath)
				}
				if hasGlob(feature) {
					matches, err := filepath.Glob(featurePath)
					if err != nil {
						add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("invalid glob %q", feature))
						continue
					}
					if len(matches) == 0 {
						add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("no matches for %q", feature))
					}
					continue
				}
				info, err := os.Stat(featurePath)
				if err != nil {
					add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("path not found at %q", feature))
				} else if info.IsDir() {
					add(fmt.Sprintf("%s.features[%d]", fieldPrefix, featureIndex), fmt.Sprintf("path %q is a directory", feature))
				}
			}
		}
	}

	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// hasGlob reports whether a path includes glob characters.
func hasGlob(value string) bool {
	return strings.ContainsAny(value, "*?[]")
}
